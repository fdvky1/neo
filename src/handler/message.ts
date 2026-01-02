import type { MessageUpsertType, WAMessage, WASocket } from "baileys";
import type { MessageWrapper, QuotedMessage } from "../types/message.js";
import { parseMention } from "../utils/index.js";

// Helper function to check if message is from Baileys
const isBaileysMessage = (id: string): boolean => {
  return (id.startsWith("BAE5") && id.length === 16) || 
         (id.startsWith("3EB0") && id.length === 12);
};

// Helper function to normalize sender ID
const normalizeSender = (sender: string): string => {
  if (sender.includes(":") && sender.endsWith("@s.whatsapp.net")) {
    return sender.split(":")[0] + "@s.whatsapp.net";
  }
  return sender;
};

// Helper function to get user ID
const getUserId = (sock: WASocket): string => {
  return sock.user?.id.split(":")[0] + "@s.whatsapp.net" || "";
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
  
  // Determine sender
  if (m.key.fromMe) {
    m.sender = getUserId(sock);
  } else if (m.isGroup) {
    m.sender = m.key.participant || m.key.participantAlt || "";
  } else {
    m.sender = m.from;
  }
  
  m.sender = normalizeSender(m.sender);
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
  
  // Determine quoted sender
  let quotedSender = contextInfo.participant || "";
  const userLid = sock.user.lid?.split(":")[0] + "@lid";
  
  if (contextInfo.participant === userLid) {
    quotedSender = sock.user.id;
  } else if (contextInfo.participant?.endsWith("@lid")) {
    const pn = await sock.signalRepository.lidMapping.getPNForLID(contextInfo.participant);
    quotedSender = pn || contextInfo.participant;
  }
  
  quotedSender = normalizeSender(quotedSender);
  const userId = getUserId(sock);
  
  const quoted: QuotedMessage = {
    mtype: quotedType,
    id: quotedId,
    from: quotedFrom,
    isBaileys: isBaileysMessage(quotedId),
    sender: quotedSender,
    fromMe: quotedSender === userId,
    text: quotedContent.text || quotedContent.caption || "",
    mentionedJid: quotedContent.contextInfo?.mentionedJid || [],
    url: quotedContent.url,
    fakeObj: {
      key: {
        remoteJid: quotedFrom,
        ...(quotedFrom.endsWith("@g.us") ? { participant: m.sender } : {}),
        fromMe: quotedSender === userId,
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
};

// Main message handler
const messageHandler = async (
  upsert: {
    messages: WAMessage[];
    type: MessageUpsertType;
    requestId?: string;
  },
  sock: WASocket
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
  console.log("Message received:", m);
};

export default messageHandler;