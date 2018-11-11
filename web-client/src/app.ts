import Terminal from "xterm/src/xterm.ts";
import * as attach from "./attach.ts";
import * as fullscreen from "xterm/src/addons/fullscreen/fullscreen.ts";
import * as fit from "xterm/src/addons/fit/fit.ts";

import "xterm/dist/xterm.css";
import "xterm/dist/addons/fullscreen/fullscreen.css";

// imports "Go"
import "./wasm_exec.js";

Terminal.applyAddon(attach);
Terminal.applyAddon(fullscreen);
Terminal.applyAddon(fit);

const go = new Go();
WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then(
  result => {
    let mod = result.module;
    let inst = result.instance;
    go.run(inst);
  }
);

const create10kbFile = (path: string, body: string): void =>
  fetch("https://up.10kb.site/" + path, {
    method: "POST",
    body: body
  })
    .then(resp => resp.text())
    .then(resp => {});

const startSession = (data: string) => {
  decode(data, (Sdp, tenKbSiteLoc, err) => {
    if (err != "") {
      console.log(err);
    }
    if (tenKbSiteLoc != "") {
      TenKbSiteLoc = tenKbSiteLoc;
    }
    pc
      .setRemoteDescription(
        new RTCSessionDescription({
          type: "offer",
          sdp: Sdp
        })
      )
      .catch(log);
    pc
      .createAnswer()
      .then(d => pc.setLocalDescription(d))
      .catch(log);
  });
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
      encode(pc.localDescription.sdp, (encoded, err) => {
        if (err != "") {
          console.log(err);
        }
        term.write(encoded);
      });
    } else {
      term.write("Waiting for connection...");
      encode(pc.localDescription.sdp, (encoded, err) => {
        if (err != "") {
          console.log(err);
        }
        create10kbFile(TenKbSiteLoc, encoded);
      });
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
