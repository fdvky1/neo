import type { WASocket } from "baileys";
import type { ContactManager } from "../utils/contact.js";

export interface SocketWrapper extends WASocket {
  // Add any additional properties or methods if needed
  contactManager: ContactManager
}