/**
 * Copyright (c) 2014, 2019 The xterm.js authors. All rights reserved.
 * @license MIT
 *
 * Implements the attach method, that attaches the terminal to a RTCDataChannel stream.
 */

import type { Terminal, IDisposable, ITerminalAddon } from '@xterm/xterm';
// import type { AttachAddon as IAttachApi } from '@xterm/addon-attach';

interface IAttachOptions {
  bidirectional?: boolean;
}

export class AttachAddon implements ITerminalAddon {
  private _rtcChannel: RTCDataChannel;
  private _bidirectional: boolean;
  private _disposables: IDisposable[] = [];

  constructor(rtcChannel: RTCDataChannel, options?: IAttachOptions) {
    this._rtcChannel = rtcChannel;
    // always set binary type to arraybuffer, we do not handle blobs
    this._rtcChannel.binaryType = 'arraybuffer';
    this._bidirectional = !(options && options.bidirectional === false);
  }

  public activate(terminal: Terminal): void {
    this._disposables.push(
      addRtcChannelListener(this._rtcChannel, 'message', ev => {
        const data: ArrayBuffer | string = ev.data;
        terminal.write(typeof data === 'string' ? data : new Uint8Array(data));
      })
    );

    if (this._bidirectional) {
      this._disposables.push(terminal.onData(data => this._sendData(data)));
      this._disposables.push(terminal.onBinary(data => this._sendBinary(data)));
    }

    this._disposables.push(addRtcChannelListener(this._rtcChannel, 'close', () => this.dispose()));
    this._disposables.push(addRtcChannelListener(this._rtcChannel, 'error', () => this.dispose()));
  }

  public dispose(): void {
    for (const d of this._disposables) {
      d.dispose();
    }
  }

  public setSize = (size: { rows: number; cols: number }) => {
    this._rtcChannel.send(JSON.stringify(["set_size", size.rows, size.cols]));
  };

  private _sendData(data: string): void {
    if (!this._checkOpenRtcChannel()) {
      return;
    }
    this._rtcChannel.send(JSON.stringify(["stdin", data]));
  }

  private _sendBinary(data: string): void {
    if (!this._checkOpenRtcChannel()) {
      return;
    }
    const buffer = new Uint8Array(data.length);
    for (let i = 0; i < data.length; ++i) {
      buffer[i] = data.charCodeAt(i) & 255;
    }
    this._rtcChannel.send(buffer);
  }

  private _checkOpenRtcChannel(): boolean {
    switch (this._rtcChannel.readyState) {
      case "open":
        return true;
      case "connecting":
        throw new Error('Attach addon was loaded before rtcChannel was open');
      case "closing":
        console.warn('Attach addon rtcChannel is closing');
        return false;
      case "closed":
        throw new Error('Attach addon rtcChannel is closed');
      default:
        throw new Error('Unexpected rtcChannel state');
    }
  }
}

function addRtcChannelListener<K extends keyof RTCDataChannelEventMap>(rtcChannel: RTCDataChannel, type: K, handler: (this: RTCDataChannel, ev: RTCDataChannelEventMap[K]) => any): IDisposable {
  rtcChannel.addEventListener(type, handler);
  return {
    dispose: () => {
      if (!handler) {
        // Already disposed
        return;
      }
      rtcChannel.removeEventListener(type, handler);
    }
  };
}
