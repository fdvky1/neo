import fs from "fs";
import path from "path";
import crypto from "crypto";
import { downloadMediaMessage as baileysDownloadMedia } from "baileys";
import { fileTypeFromBuffer } from "file-type";
import webpmux from "node-webpmux";
import type { WASocket } from "baileys";
import type { MessageWrapper } from "../types/message.js";
import logger from "../lib/logger.js";

const { Image } = webpmux;

const parseMention = (text = "") => 
  [...text.toString().matchAll(/@([0-9]{5,16}|0)/g)].map((v) => v[1] + "@s.whatsapp.net");

/**
 * Generate cryptographically secure random string
 * @param length - Length of the random string
 * @returns Random string
 */
const random = (length: number = 16): string => {
  return crypto.randomBytes(Math.ceil(length / 2))
    .toString('hex')
    .slice(0, length);
};

/**
 * Create EXIF data for WhatsApp stickers
 * @param packname - Sticker pack name (default from env)
 * @param author - Sticker pack author (default from env)
 * @returns EXIF buffer
 */
const createExif = (
  packname: string = process.env.STICKER_PACK_NAME || "NEO Bot",
  author: string = process.env.STICKER_PACK_AUTHOR || "NEO Whatsapp Bot"
): Buffer => {
  const json = {
    "sticker-pack-id": `neo-bot.whatsapp`,
    "sticker-pack-name": packname,
    "sticker-pack-publisher": author,
    emojis: ["👋"],
  };

  const exifAttr = Buffer.from([
    0x49, 0x49, 0x2a, 0x00, 0x08, 0x00, 0x00, 0x00, 0x01, 0x00, 0x41, 0x57,
    0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x16, 0x00, 0x00, 0x00,
  ]);

  const jsonBuffer = Buffer.from(JSON.stringify(json), "utf8");
  const exif = Buffer.concat([exifAttr, jsonBuffer]);
  exif.writeUIntLE(jsonBuffer.length, 14, 4);

  return exif;
};

// Cache default EXIF to avoid recreating it repeatedly
const defaultExif = createExif();

/**
 * Modify WebP image to add EXIF data for WhatsApp stickers
 * @param webpMedia - WebP image buffer
 * @param options - Pack name and author options
 * @returns Modified WebP buffer with EXIF
 */
const modifyExif = async (
  webpMedia: Buffer,
  options: {
    packname?: string;
    author?: string;
  } = {}
): Promise<Buffer> => {
  const img = new Image();
  await img.load(webpMedia);

  // Use cached default EXIF or create new one if custom values provided
  img.exif = 
    !options.packname && !options.author
      ? defaultExif
      : createExif(options.packname, options.author);

  return await img.save(null);
};

/**
 * Download media from a message
 * @param sock - WASocket instance
 * @param m - Message wrapper containing media
 * @param result - Either 'buffer' to return buffer, or filename to save to disk
 * @returns Buffer if result is 'buffer', otherwise returns the saved file path
 */
const downloadMediaMessage = async (
  sock: WASocket,
  m: MessageWrapper,
  result: string = "buffer"
): Promise<Buffer | string> => new Promise(async (resolve, reject) => {
    try {
      // Download media as buffer
      let buffer = await baileysDownloadMedia(
        m,
        "buffer",
        {},
        { 
          reuploadRequest: sock.updateMediaMessage,
          logger
        }
      );

      // If result is buffer, return buffer directly
      if (result === "buffer") {
        return resolve(buffer);
      }

      // Ensure tmp directory exists
      const tmpDir = path.join(process.cwd(), "tmp");
      if (!fs.existsSync(tmpDir)) {
        fs.mkdirSync(tmpDir, { recursive: true });
      }

      // Detect file extension from buffer
      const fileType = await fileTypeFromBuffer(buffer);
      const ext = fileType?.ext || "bin";
      
      // Create file path
      const fileName = path.join(tmpDir, `${result}.${ext}`);
      
      // Write file to disk
      fs.writeFileSync(fileName, buffer);
      
      // Clear buffer from memory
      buffer = null as any;
      
      resolve(fileName);
    } catch (error) {
      reject(error);
    }
  });

export { parseMention, downloadMediaMessage, random, createExif, modifyExif };