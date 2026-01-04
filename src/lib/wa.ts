import makeWASocket, { useMultiFileAuthState, makeCacheableSignalKeyStore, Browsers, DisconnectReason } from 'baileys'
import logger from './logger.js'
import messageHandler from '../handler/message.js'
import qrcode from 'qrcode-terminal'
import { ContactManager } from '../utils/contact.js'
import type { SocketWrapper } from '../types/socket.js'

const init = async (clientId: string) => {
    const {state, saveCreds} = await useMultiFileAuthState(`auth_info/${clientId}`)
    
    // Initialize contact manager
    
    
    const sock = makeWASocket({
        logger,
        auth: {
            creds: state.creds,
            keys: makeCacheableSignalKeyStore(state.keys, logger)
        },
        // printQRInTerminal: true,
        markOnlineOnConnect: true,
        syncFullHistory: false,
        // options: {
        //     headers: {
        //         'User-Agent': 'Mozilla/5.0 (Linux; Android 12; vivo 1915) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Mobile Safari/537.36'
        //     }
        // }
        browser: Browsers.macOS('NEO')
    }) as SocketWrapper
    
    sock.contactManager = new ContactManager(clientId);
    sock.contactManager.load();
    sock.contactManager.startAutoSave();

    sock.ev.on('connection.update', (update) => {
        if(!!update.qr){
            qrcode.generate(update.qr, { small: true })
        }
        const {connection, lastDisconnect} = update
        if(connection === 'close') {
            // Save contacts and stop auto-save before closing
            sock.contactManager.save();
            sock.contactManager.stopAutoSave();
            
            const shouldReconnect = (lastDisconnect?.error as any)?.output?.statusCode !== DisconnectReason.loggedOut
            console.log('connection closed due to ', lastDisconnect?.error, ', reconnecting ', shouldReconnect)
            // reconnect if not logged out
            if(shouldReconnect) {
                init(clientId)
            }
        } else if(connection === 'open') {
            console.log('opened connection')
        }
    })

    sock.ev.on('creds.update', saveCreds)

    sock.ev.on('messages.upsert', async (m) => {
        messageHandler(m, sock)
    })
    // return sock
}

export default init