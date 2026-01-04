import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import type { Command } from "../types/cmd.js";

// Get current directory for ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Custom Collection class with type safety
class Collection<K, V> extends Map<K, V> {
  find(
    fn: (value: V, key: K, collection: this) => boolean,
    thisArg?: any
  ): V | undefined {
    if (typeof thisArg !== "undefined") fn = fn.bind(thisArg);
    for (const [key, val] of this) {
      if (fn(val, key, this)) return val;
    }
    return undefined;
  }

  filter(fn: (value: V, key: K, collection: this) => boolean): Collection<K, V> {
    const results = new Collection<K, V>();
    for (const [key, val] of this) {
      if (fn(val, key, this)) results.set(key, val);
    }
    return results;
  }

  map<T>(fn: (value: V, key: K, collection: this) => T): T[] {
    const results: T[] = [];
    for (const [key, val] of this) {
      results.push(fn(val, key, this));
    }
    return results;
  }
}

// Commands collection
const commands = new Collection<string, Command>();

// Load commands from subdirectories
const loadCommands = async () => {
  const cmdsPath = path.join(__dirname);
  
  try {
    const entries = fs.readdirSync(cmdsPath, { withFileTypes: true });
    
    // Get only directories
    const directories = entries
      .filter((entry) => entry.isDirectory())
      .map((entry) => entry.name);

    for (const dir of directories) {
      const dirPath = path.join(cmdsPath, dir);
      const files = fs.readdirSync(dirPath).filter(
        (file) => file.endsWith(".js")
      );

      for (const file of files) {
        try {
          const filePath = path.join(dirPath, file);
          const module = await import(filePath);
          const command: Command = module.default;

          if (command && command.name) {
            commands.set(command.name, {
              ...command,
              category: dir,
            });
            console.log(`Loaded command: ${command.name} from ${dir}/${file}`);
          }
        } catch (error) {
          console.error(`Error loading command from ${dir}/${file}:`, error);
        }
      }
    }
  } catch (error) {
    console.error("Error loading commands:", error);
  }
};

// Load commands on module initialization
await loadCommands();

export { commands, Collection };
// export type { Command, CommandContext } from "../types/cmd";
export default commands;
