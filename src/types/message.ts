import type { WAMessage } from "baileys";

export interface QuotedMessage {
  mtype: string;
  id: string;
  from: string;
  isBaileys: boolean;
  sender: string;
  fromMe: boolean;
  text: string;
  mentionedJid: string[];
  url?: string;
  fakeObj: WAMessage;
  reply: (text: string, chatId?: string) => Promise<any>;
  forward: (chatId?: string) => Promise<any>;
  delete: () => Promise<any>;
}

export interface MessageWrapper extends WAMessage {
  id: string;
  isBaileys: boolean;
  from: string;
  fromMe: boolean;
  isGroup: boolean;
  sender: string;
  mtype: string;
  msg: any;
  quoted: QuotedMessage | null;
  mentionedJid: string[];
  text: string;
  reply: (text: string, chatId?: string, options?: any) => Promise<any>;
  forward: (chatId?: string) => Promise<any>;
}
