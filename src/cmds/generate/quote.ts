import type { WASocket } from "baileys";
import type { MessageWrapper } from "../../types/message.js";
import type { Command, CommandContext } from "../../types/cmd.js";
import { modifyExif } from "../../utils/index.js";

// Quote API request interface
interface QuoteRequest {
  type: "quote";
  format: "webp";
  isWaSticker: true;
  backgroundColor: string;
  width: number;
  height: number;
  scale: number;
  messages: Array<{
    entities: any[];
    media?: {
      base64: string;
    };
    avatar: boolean;
    from: {
      id: string | undefined;
      name: string;
      photo: {
        url?: string;
        createImageLatters?: boolean;
      };
    };
    text: string;
    replyMessage: {
      chatId?: string;
      name?: string;
      text?: string;
    };
  }>;
}

// Quote API response interface
interface QuoteResponse {
  ok: boolean;
  result: {
    image: string; // base64 string
  };
}

const command: Command = {
  name: "quote",
  description: "Generate a quote sticker from text or media",
  aliases: ["q", "qc"],
  async execute(sock, m, { args, usedPrefix, command }) {
    let media: Buffer | undefined;

    // Check if current message has media
    if (["stickerMessage", "imageMessage"].includes(m.mtype)) {
      media = await m.download() as Buffer;
    } 
    // Check if quoted message has media and no args provided
    else if (
      m.quoted &&
      ["stickerMessage", "imageMessage"].includes(m.quoted.mtype) &&
      args.length === 0
    ) {
      media = await m.quoted.download() as Buffer;
    }

    // Validate input
    const hasText = (m.quoted?.text || args.join(" ")).length > 0;
    if (!hasText && !media) {
      return m.reply(
        `Send command ${usedPrefix}${command} with text or reply to a message`
      );
    }

    // Get profile picture
    let profilePicture: string | null = null;
    const targetSender = args.length > 0 ? m.sender : m.quoted?.sender || m.sender;
    
    try {
      const ppUrl = await sock.profilePictureUrl(targetSender, "image");
      profilePicture = ppUrl || null;
    } catch (error) {
      // Profile picture not available, will use createImageLatters
    }

    // Prepare text and sender info
    const text = args.length > 0 ? args.join(" ") : m.quoted?.text || "";
    const senderId = (targetSender || m.sender).split("@")[0];
    const senderName = args.length > 0 
      ? m.pushName || `+${senderId}`
      : m.quoted?.sender
        ? `${sock.contactManager.getUser(m.quoted.sender).notify || '+' + m.quoted.senderNumber.split("@")[0]}`
        : m.pushName || `+${senderId}`;

    // Determine optimal size based on content
    const size = media ? 720 : text.length < 170 ? 520 : 1024;

    // Build reply message if applicable
    const replyMessage: { chatId?: string; name?: string; text?: string } = {};
    if (args.length > 0 && m.quoted?.text && m.quoted.text.length > 0 && m.quoted.sender) {
      const quotedSenderId = m.quoted.sender.split("@")[0];
      if (quotedSenderId) {
        // Use contactManager to retrieve pushName from quoted message
        const quotedUser = sock.contactManager.getUser(m.quoted.sender);
        const quotedName = quotedUser.notify || quotedUser.name || `+${quotedSenderId}`;
        
        replyMessage.chatId = quotedSenderId;
        replyMessage.name = quotedName;
        replyMessage.text = m.quoted.text;
      }
    }

    // Build quote request payload
    const payload: QuoteRequest = {
      type: "quote",
      format: "webp",
      isWaSticker: true,
      backgroundColor: "#1b1429",
      width: size,
      height: size,
      scale: 2,
      messages: [
        {
          entities: [],
          ...(media ? { media: { base64: media.toString("base64") } } : {}),
          avatar: true,
          from: {
            id: senderId,
            name: senderName,
            photo: profilePicture
              ? { url: profilePicture }
              : { createImageLatters: true },
          },
          text,
          replyMessage,
        },
      ],
    };

    try {
      // Call quote API using fetch
      const quoteUrl = process.env.QUOTE_URL! //|| "https://bot.lyo.su/quote/generate";
      const response = await fetch(quoteUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error(`API request failed with status ${response.status}`);
      }

      const data: QuoteResponse = await response.json();

      if (!data.ok || !data.result?.image) {
        throw new Error("Invalid response from quote API");
      }

      // Convert base64 to buffer and add EXIF
      const imageBuffer = Buffer.from(data.result.image, "base64");
      const sticker = await modifyExif(imageBuffer);

      // Send sticker
      return sock.sendMessage(
        m.from,
        { sticker },
        { quoted: m }
      );
    } catch (error) {
      console.error("Quote generation error:", error);
      return m.reply(
        `Failed to generate quote: ${error instanceof Error ? error.message : "Unknown error"}`
      );
    }
  },
};

export default command;
