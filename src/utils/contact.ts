import type { Contact } from "baileys";
import fs from "fs";
import path from "path";

class ContactManager {
  private users: Map<string, Contact>;
  private clientId: string;
  private saveInterval: NodeJS.Timeout | undefined;

  constructor(clientId: string) {
    this.users = new Map<string, Contact>();
    this.clientId = clientId;
    this.saveInterval = undefined;
  }

  /**
   * Set or update a contact
   */
  setUser(contact: Contact): void {
    this.users.set(contact.id, contact);
    // console.log(this.users)
  }

  /**
   * Get a contact by ID, creates a new one if not exists
   */
  getUser(id: string): Contact {
    if (!this.users.has(id)) {
      this.users.set(id, { id });
    }
    return this.users.get(id)!;
  }

  /**
   * Get all users
   */
  getAllUsers(): Map<string, Contact> {
    return this.users;
  }

  /**
   * Save contacts to JSON file
   */
  save(): void {
    try {
      const contactsPath = path.join(process.cwd(), `auth_info/${this.clientId}`);
      
      // Ensure directory exists
      if (!fs.existsSync(contactsPath)) {
        fs.mkdirSync(contactsPath, { recursive: true });
      }
      
      // Convert Map to array for JSON serialization
      const contactsArray = Array.from(this.users.entries()).map(([id, contact]) => contact);
      
      // Write to file
      const filePath = path.join(contactsPath, "contact.json");
      fs.writeFileSync(filePath, JSON.stringify(contactsArray, null, 2));
      
      console.log(`Saved ${contactsArray.length} contacts to ${filePath}`);
    } catch (error) {
      console.error("Error saving contacts:", error);
    }
  }

  /**
   * Load contacts from JSON file
   */
  load(): void {
    try {
      const filePath = path.join(process.cwd(), `auth_info/${this.clientId}/contact.json`);
      
      if (!fs.existsSync(filePath)) {
        console.log("No existing contacts file found");
        return;
      }
      
      const data = fs.readFileSync(filePath, "utf-8");
      const contactsArray: Contact[] = JSON.parse(data);
      
      // Load contacts into Map
      contactsArray.forEach(contact => {
        this.users.set(contact.id, contact);
      });
      
      console.log(`Loaded ${contactsArray.length} contacts from ${filePath}`);
    } catch (error) {
      console.error("Error loading contacts:", error);
    }
  }

  /**
   * Start auto-save interval (every 10 minutes)
   */
  startAutoSave(): void {
    if (this.saveInterval) {
      clearInterval(this.saveInterval);
    }
    
    this.saveInterval = setInterval(() => {
      this.save();
    }, 10 * 60 * 1000); // 10 minutes
    
    console.log("Contact auto-save started (every 10 minutes)");
  }

  /**
   * Stop auto-save interval
   */
  stopAutoSave(): void {
    if (this.saveInterval) {
      clearInterval(this.saveInterval);
      this.saveInterval = undefined;
      console.log("Contact auto-save stopped");
    }
  }

  /**
   * Get the count of contacts
   */
  count(): number {
    return this.users.size;
  }
}

export { ContactManager };
export type { Contact };