// Client-side cryptography utilities for shebang.run

const ShebangCrypto = {
    async importPrivateKey(pemKey) {
        let pemContents = pemKey.trim()
            .replace(/-----BEGIN RSA PRIVATE KEY-----/g, "")
            .replace(/-----END RSA PRIVATE KEY-----/g, "")
            .replace(/-----BEGIN PRIVATE KEY-----/g, "")
            .replace(/-----END PRIVATE KEY-----/g, "")
            .replace(/\s/g, "");
        
        const binaryDer = this.base64ToArrayBuffer(pemContents);
        
        try {
            return await window.crypto.subtle.importKey(
                "pkcs8",
                binaryDer,
                { name: "RSA-OAEP", hash: "SHA-256" },
                false,
                ["decrypt"]
            );
        } catch (e) {
            throw new Error("Private key must be in PKCS8 format");
        }
    },

    async unwrapKey(wrappedKeyArray, privateKey) {
        // Convert to Uint8Array properly
        let wrappedKeyBuffer;
        
        if (wrappedKeyArray instanceof Uint8Array) {
            wrappedKeyBuffer = wrappedKeyArray;
        } else if (Array.isArray(wrappedKeyArray)) {
            wrappedKeyBuffer = new Uint8Array(wrappedKeyArray);
        } else if (typeof wrappedKeyArray === 'object' && wrappedKeyArray.data) {
            // Handle {type: 'Buffer', data: [...]} format
            wrappedKeyBuffer = new Uint8Array(wrappedKeyArray.data);
        } else {
            throw new Error('Invalid wrapped key format');
        }
        
        console.log('Unwrapping key, buffer length:', wrappedKeyBuffer.length);
        
        try {
            const unwrappedKey = await window.crypto.subtle.decrypt(
                { name: "RSA-OAEP" },
                privateKey,
                wrappedKeyBuffer
            );
            
            return new Uint8Array(unwrappedKey);
        } catch (e) {
            console.error('RSA decrypt error:', e);
            throw new Error('Failed to unwrap key. Wrong private key or corrupted data.');
        }
    },

    async decryptContent(encryptedDataArray, symmetricKey) {
        await window.sodium.ready;
        const sodium = window.sodium;
        
        const data = encryptedDataArray instanceof Uint8Array 
            ? encryptedDataArray 
            : new Uint8Array(encryptedDataArray);
        
        const nonceSize = sodium.crypto_aead_xchacha20poly1305_ietf_NPUBBYTES;
        const nonce = data.slice(0, nonceSize);
        const ciphertext = data.slice(nonceSize);
        
        try {
            const decrypted = sodium.crypto_aead_xchacha20poly1305_ietf_decrypt(
                null,
                ciphertext,
                null,
                nonce,
                symmetricKey
            );
            
            return this.arrayBufferToString(decrypted);
        } catch (e) {
            console.error('Sodium decrypt error:', e);
            throw new Error("Decryption failed. Wrong key or corrupted data.");
        }
    },

    base64ToArrayBuffer(base64) {
        const binaryString = window.atob(base64);
        const bytes = new Uint8Array(binaryString.length);
        for (let i = 0; i < binaryString.length; i++) {
            bytes[i] = binaryString.charCodeAt(i);
        }
        return bytes.buffer;
    },

    arrayBufferToString(buffer) {
        return new TextDecoder().decode(buffer);
    }
};

window.ShebangCrypto = ShebangCrypto;
