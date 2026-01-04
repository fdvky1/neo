import type { MessageUpsertType, WAMessage, WASocket } from "baileys";
import type { MessageWrapper, QuotedMessage } from "../types/message.js";
import { parseMention, downloadMediaMessage } from "../utils/index.js";
import commands from "../cmds/index.js";
import type { ContactManager } from "../utils/contact.js";
import type { SocketWrapper } from "../types/socket.js";

// Helper function to check if message is from Baileys
const isBaileysMessage = (id: string): boolean => {
  return (id.startsWith("BAE5") && id.length === 16) || 
         (id.startsWith("3EB0") && id.length === 12);
};

// Helper function to normalize @lid format (remove :XX suffix if exists)
const normalizeLid = (lid: string): string => {
  if (!lid) return lid;
  if (lid.includes(":") && lid.endsWith("@lid")) {
    return lid.split(":")[0] + "@lid";
  }
  return lid;
};

// Helper function to normalize phone number format (remove :XX suffix if exists)
const normalizeNumber = (number: string): string => {
  if (!number) return number;
  if (number.includes(":") && number.endsWith("@s.whatsapp.net")) {
    return number.split(":")[0] + "@s.whatsapp.net";
  }
  return number;
};

// Helper function to extract message type
const getMessageType = (message: any): string => {
  const keys = Object.keys(message).filter(
    (v) => v !== "senderKeyDistributionMessage" && v !== "messageContextInfo"
  );
  return keys[0] || "conversation";
};

// Parse basic message properties
const parseBasicMessage = (m: MessageWrapper, sock: WASocket): void => {
  if (!sock.user) return;

  m.id = m.key.id || "";
  m.isBaileys = isBaileysMessage(m.id);
  m.from = m.key.remoteJid || "";
  m.fromMe = m.key.fromMe || false;
  m.isGroup = m.from.endsWith("@g.us");
  
  // Determine sender and senderNumber based on chat type
  if (m.key.fromMe) {
    // Message from bot itself
    m.sender = normalizeLid(sock.user.lid!); // @lid format
    m.senderNumber = normalizeNumber(sock.user.id); // @s.whatsapp.net format
  } else if (m.isGroup) {
    // Group message - use participant fields
    m.sender = normalizeLid(m.key.participant || ""); // @lid format
    m.senderNumber = normalizeNumber(m.key.participantAlt || m.sender); // @s.whatsapp.net format
  } else {
    // Private message - use remoteJid fields
    m.sender = normalizeLid(m.key.remoteJid || ""); // @lid format
    m.senderNumber = normalizeNumber(m.key.remoteJidAlt || m.sender); // @s.whatsapp.net format
  }
};

// Parse message content
const parseMessageContent = (m: MessageWrapper): void => {
  if (!m.message) return;
  
  m.mtype = getMessageType(m.message);
  m.msg = (m.message as any)[m.mtype];
  
  // Handle string messages
  if (typeof m.msg === "string") {
    m.msg = { text: m.msg };
  }
  
  // Handle ephemeral and view once messages
  if (["ephemeralMessage", "viewOnceMessage"].includes(m.mtype) && m.msg?.message) {
    const innerKeys = Object.keys(m.msg.message);
    m.mtype = innerKeys[0] || "conversation";
    m.msg = m.msg.message[m.mtype];
  }
  
  m.text = m.msg?.text || m.msg?.caption || "";
  m.mentionedJid = m.msg?.contextInfo?.mentionedJid || [];
};

// Parse quoted message
const parseQuotedMessage = async (m: MessageWrapper, sock: WASocket): Promise<void> => {
  if (!m.msg?.contextInfo?.quotedMessage || !sock.user) {
    m.quoted = null;
    return;
  }

  const contextInfo = m.msg.contextInfo;
  const quotedKeys = Object.keys(contextInfo.quotedMessage);
  let quotedType = quotedKeys[0] || "conversation";
  let quotedContent = (contextInfo.quotedMessage as any)[quotedType];
  
  // Handle string quoted messages
  if (typeof quotedContent === "string") {
    quotedContent = { text: quotedContent };
  }
  
  // Handle view once quoted messages
  if (quotedType === "viewOnceMessage" && quotedContent?.message) {
    const innerKeys = Object.keys(quotedContent.message);
    quotedType = innerKeys[0] || "conversation";
    quotedContent = quotedContent.message[quotedType];
  }
  
  const quotedId = contextInfo.stanzaId || "";
  const quotedFrom = contextInfo.remoteJid || m.from;
  
  // For quoted messages, contextInfo.participant is always in PN format (@s.whatsapp.net)
  let quotedSenderNumber = normalizeNumber(contextInfo.participant || "");
  let quotedSender = "";
  
  // Check if quoted message is from bot itself
  const userLid = normalizeLid(sock.user.lid || "");
  const userNumber = normalizeNumber(sock.user.id);
  
  if (quotedSenderNumber === userNumber) {
    // Quoted message from bot itself
    quotedSender = userLid;
  } else {
    // Try to get LID from phone number
    try {
      const lid = await sock.signalRepository.lidMapping.getLIDForPN(quotedSenderNumber);
      quotedSender = normalizeLid(lid || quotedSenderNumber);
    } catch (error) {
      // If can't get LID, use phone number as fallback
      quotedSender = quotedSenderNumber;
    }
  }
  
  const quoted: QuotedMessage = {
    mtype: quotedType,
    id: quotedId,
    from: quotedFrom,
    isBaileys: isBaileysMessage(quotedId),
    sender: quotedSender,
    senderNumber: quotedSenderNumber,
    fromMe: quotedSenderNumber === userNumber,
    text: quotedContent.text || quotedContent.caption || "",
    mentionedJid: quotedContent.contextInfo?.mentionedJid || [],
    url: quotedContent.url,
    fakeObj: {
      key: {
        remoteJid: quotedFrom,
        ...(quotedFrom.endsWith("@g.us") ? { participant: quotedSender } : {}),
        fromMe: quotedSenderNumber === userNumber,
        id: quotedId,
      },
      message: contextInfo.quotedMessage,
    },
    reply: (text: string, chatId?: string) =>
      sock.sendMessage(
        chatId || quotedFrom,
        { text, mentions: parseMention(text) },
        { quoted: quoted.fakeObj }
      ),
    forward: (chatId?: string) =>
      sock.sendMessage(chatId || quotedFrom, { forward: quoted.fakeObj }),
    delete: () =>
      sock.sendMessage(quotedFrom, { delete: { ...quoted.fakeObj.key } as any }),
    download: (result: string = "buffer") =>
      downloadMediaMessage(sock, quoted.fakeObj as MessageWrapper, result),
  };
  
  m.quoted = quoted;
};

// Add message methods
const addMessageMethods = (m: MessageWrapper, sock: WASocket): void => {
  m.reply = (text: string, chatId?: string, options?: any) =>
    sock.sendMessage(
      chatId || m.from,
      { text, mentions: parseMention(text) },
      { quoted: m, ...options }
    );
    
  m.forward = (chatId?: string) =>
    sock.sendMessage(chatId || m.from, { forward: m });
    
  m.download = (result: string = "buffer") =>
    downloadMediaMessage(sock, m, result);
};

// Main message handler
const messageHandler = async (
  upsert: {
    messages: WAMessage[];
    type: MessageUpsertType;
    requestId?: string;
  },
  sock: SocketWrapper
) => {
  // Early return checks
  if (upsert.type !== "notify") return;
  if (!sock.user) return;
  
  const firstMessage = upsert.messages[0];
  if (!firstMessage?.message) return;
  if (!firstMessage.message["protocolMessage"] && 
      firstMessage.key?.remoteJid?.endsWith("@broadcast")) return;
  // Parse message
  const m = firstMessage as MessageWrapper;
  
  parseBasicMessage(m, sock);
  parseMessageContent(m);
  await parseQuotedMessage(m, sock);
  addMessageMethods(m, sock);
  
  // TODO: Add your command/message handling logic here
  // console.log("Message received:", m);
  if (m.isBaileys) return;

  // cache user contact info
  sock.contactManager.setUser({
    id: m.sender,
    phoneNumber: m.senderNumber || m.sender,
    ...(!!m.pushName && m.pushName.length > 0 ? { notify: m.pushName } : {})
  })

  if(process.env.BOT_MODE === "self_only" && !m.fromMe) return;
  if(process.env.BOT_MODE === "pub_only" && m.fromMe) return;

  const prefix = m.text.split("")[0];
  let args = m.text.trim().split(/ +/);
  const usedPrefix = prefix && /^[.#$&\/\\?!]/.test(prefix) ? prefix.match(/^[.#$&\/\\?!]/gi)![0] : "!";
  const command =  m.text.startsWith(usedPrefix) ? m.text.slice(1).trim().split(/ +/).shift()!.toLowerCase() : undefined;
  if (m.text.startsWith(usedPrefix)) args = args.slice(1);
  if (args[0] === command) args = args.slice(1);
  const cmd = command ? commands.get(command) || commands.find((c) => (c.aliases || []).includes(command)) : false;
  if(cmd){
    console.log(`Executing command: ${usedPrefix}${command} by ${m.sender}/${m.senderNumber}`);
    cmd.execute(sock, m, {args, usedPrefix, command: command!});
  }
};

export default messageHandler;