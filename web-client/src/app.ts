import Terminal from "xterm/src/xterm.ts";
import * as attach from "./attach.ts";
import * as fullscreen from "xterm/src/addons/fullscreen/fullscreen.ts";
import * as fit from "xterm/src/addons/fit/fit.ts";
import pako from "pako";
import aesjs from "aes-js";

import "xterm/dist/xterm.css";
import "xterm/dist/addons/fullscreen/fullscreen.css";
import bs58 from "bs58";
import { Buffer } from "safe-buffer";
window.pako = pako;

Terminal.applyAddon(attach);
Terminal.applyAddon(fullscreen);
Terminal.applyAddon(fit);

const encodeOffer = (data: string) =>
  bs58.encode(Buffer.from(pako.deflate(JSON.stringify({ Sdp: data }))));

const decodeOffer = (data: string): string =>
  JSON.parse(pako.inflate(bs58.decode(data), { to: "string" }));

const create10kbFile = (path: string, body: string): void =>
  fetch("https://up.10kb.site/" + path, {
    method: "POST",
    body: body
  })
    .then(resp => resp.text())
    .then(resp => {});

const startSession = (data: string) => {
  const sessionDesc = decodeOffer(data);
  if (sessionDesc.TenKbSiteLoc != "") {
    TenKbSiteLoc = sessionDesc.TenKbSiteLoc;
  }
  pc
    .setRemoteDescription(
      new RTCSessionDescription({
        type: "offer",
        sdp: sessionDesc.Sdp
      })
    )
    .catch(log);
  pc
    .createAnswer()
    .then(d => pc.setLocalDescription(d))
    .catch(log);
};

let TenKbSiteLoc = null;

const term = new Terminal();
term.open(document.getElementById("terminal"));
term.toggleFullScreen();
term.fit();
window.onresize = () => {
  term.fit();
};
term.write("Welcome to the WebRTTY web client.\n\r");

let pc = new RTCPeerConnection({
  iceServers: [
    {
      urls: "stun:stun.l.google.com:19302"
    }
  ]
});

let log = msg => {
  term.write(msg + "\n\r");
};

let sendChannel = pc.createDataChannel("data");
sendChannel.onclose = () => console.log("sendChannel has closed");
sendChannel.onopen = () => {
  term.reset();
  term.terminadoAttach(sendChannel);
  sendChannel.send(JSON.stringify(["set_size", term.rows, term.cols]));
  console.log("sendChannel has opened");
};
// sendChannel.onmessage = e => {}

pc.onsignalingstatechange = e => log(pc.signalingState);
pc.oniceconnectionstatechange = e => log(pc.iceConnectionState);
pc.onicecandidate = event => {
  if (event.candidate === null) {
    if (TenKbSiteLoc == null) {
      term.write(
        "Answer created. Send the following answer to the host:\n\r\n\r"
      );
      term.write(encodeOffer(pc.localDescription.sdp));
    } else {
      term.write("Waiting for connection...");
      create10kbFile(TenKbSiteLoc, encodeOffer(pc.localDescription.sdp));
    }
  }
};

pc.onnegotiationneeded = e => console.log(e);

window.sendMessage = () => {
  let message = document.getElementById("message").value;
  if (message === "") {
    return alert("Message must not be empty");
  }

  sendChannel.send(message);
};

let firstInput: boolean = false;
const urlData = window.location.hash.substr(1);
console.log(urlData);
if (urlData != "") {
  try {
    startSession(urlData);
    firstInput = true;
  } catch (err) {
    console.log(err);
  }
}

if (firstInput == false) {
  term.write("Run webrtty and paste the offer message below:\n\r");
}

term.on("data", data => {
  if (!firstInput) {
    term.reset();
    try {
      startSession(data);
    } catch (err) {
      console.log(err);
      term.write(`There was an error with the offer: ${data}\n\r`);
      term.write("Try entering the message again: ");
      return;
    }
    firstInput = true;
  }
});
