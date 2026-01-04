import type { WASocket } from "baileys";
import type { MessageWrapper } from "./message";
import type { ContactManager } from "../utils/contact";
import type { SocketWrapper } from "./socket.ts";

// Command context interface
export interface CommandContext {
  args: string[];
  usedPrefix: string;
  command: string;
}

// Command interface
export interface Command {
  name: string;
  description?: string;
  aliases?: string[];
  category?: string;
  permissions?: string[];
  execute: (
    sock: SocketWrapper,
    m: MessageWrapper,
    context: CommandContext
  ) => Promise<void> | void;
}
