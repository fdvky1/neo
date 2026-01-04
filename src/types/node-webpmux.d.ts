declare module "node-webpmux" {
  export class Image {
    load(buffer: Buffer): Promise<void>;
    save(path: string | null): Promise<Buffer>;
    exif: Buffer;
  }
  
  const webpmux: {
    Image: typeof Image;
  };
  
  export default webpmux;
}
