/**
 * Copyright (c) 2016 The xterm.js authors. All rights reserved.
 * @license MIT
 *
 * This module provides methods for attaching a terminal to a terminado
 * WebSocket stream.
 */

import { Terminal } from "xterm";
import { ITerminadoAddonTerminal } from "./Interfaces";

/**
 * Attaches the given terminal to the given socket.
 *
 * @param term The terminal to be attached to the given socket.
 * @param socket The socket to attach the current terminal.
 * @param bidirectional Whether the terminal should send data to the socket as well.
 * @param buffered Whether the rendering of incoming data should happen instantly or at a maximum
 * frequency of 1 rendering per 10ms.
 */
export function terminadoAttach(
  term: Terminal,
  socket: WebSocket,
  bidirectional: boolean,
  buffered: boolean
): void {
  const addonTerminal = <ITerminadoAddonTerminal>term;
  bidirectional = typeof bidirectional === "undefined" ? true : bidirectional;
  addonTerminal.__socket = socket;

  addonTerminal.__flushBuffer = () => {
    addonTerminal.write(addonTerminal.__attachSocketBuffer);
    addonTerminal.__attachSocketBuffer = null;
  };

  addonTerminal.__pushToBuffer = (data: string) => {
    if (addonTerminal.__attachSocketBuffer) {
      addonTerminal.__attachSocketBuffer += data;
    } else {
      addonTerminal.__attachSocketBuffer = data;
      setTimeout(addonTerminal.__flushBuffer, 10);
    }
  };

  let myTextDecoder: any;
  addonTerminal.__getMessage = (ev: MessageEvent) => {
    let str: string;

    if (typeof ev.data === "object") {
      if (!myTextDecoder) {
        myTextDecoder = new TextDecoder();
      }
      if (ev.data instanceof ArrayBuffer) {
        str = myTextDecoder.decode(ev.data);
        displayData(str);
      } else {
        const fileReader = new FileReader();

        fileReader.addEventListener("load", () => {
          str = myTextDecoder.decode(fileReader.result);
          displayData(str);
        });
        fileReader.readAsArrayBuffer(ev.data);
      }
    } else if (typeof ev.data === "string") {
      displayData(ev.data);
    } else {
      throw Error(`Cannot handle "${typeof ev.data}" websocket message.`);
    }
  };

  /**
   * Push data to buffer or write it in the terminal.
   * This is used as a callback for FileReader.onload.
   *
   * @param str String decoded by FileReader.
   * @param data The data of the EventMessage.
   */
  function displayData(str?: string, data?: string): void {
    if (buffered) {
      addonTerminal.__pushToBuffer(str || data);
    } else {
      addonTerminal.write(str || data);
    }
  }

  addonTerminal.__sendData = (data: string) => {
    socket.send(JSON.stringify(["stdin", data]));
  };

  addonTerminal.__setSize = (size: { rows: number; cols: number }) => {
    socket.send(JSON.stringify(["set_size", size.rows, size.cols]));
  };

  socket.addEventListener("message", addonTerminal.__getMessage);

  if (bidirectional) {
    addonTerminal.on("data", addonTerminal.__sendData);
  }
  addonTerminal.on("resize", addonTerminal.__setSize);

  socket.addEventListener("close", () =>
    terminadoDetach(addonTerminal, socket)
  );
  socket.addEventListener("error", () =>
    terminadoDetach(addonTerminal, socket)
  );
}

/**
 * Detaches the given terminal from the given socket
 *
 * @param term The terminal to be detached from the given socket.
 * @param socket The socket from which to detach the current terminal.
 */
export function terminadoDetach(term: Terminal, socket: WebSocket): void {
  const addonTerminal = <ITerminadoAddonTerminal>term;
  addonTerminal.off("data", addonTerminal.__sendData);

  socket = typeof socket === "undefined" ? addonTerminal.__socket : socket;

  if (socket) {
    socket.removeEventListener("message", addonTerminal.__getMessage);
  }

  delete addonTerminal.__socket;
}

export function apply(terminalConstructor: typeof Terminal): void {
  /**
   * Attaches the current terminal to the given socket
   *
   * @param socket - The socket to attach the current terminal.
   * @param bidirectional - Whether the terminal should send data to the socket as well.
   * @param buffered - Whether the rendering of incoming data should happen instantly or at a
   * maximum frequency of 1 rendering per 10ms.
   */
  (<any>terminalConstructor.prototype).terminadoAttach = function(
    socket: WebSocket,
    bidirectional: boolean,
    buffered: boolean
  ): void {
    return terminadoAttach(this, socket, bidirectional, buffered);
  };

  /**
   * Detaches the current terminal from the given socket.
   *
   * @param socket The socket from which to detach the current terminal.
   */
  (<any>terminalConstructor.prototype).terminadoDetach = function(
    socket: WebSocket
  ): void {
    return terminadoDetach(this, socket);
  };
}
