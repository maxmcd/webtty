import { Terminal } from "@xterm/xterm";
import { AttachAddon } from "./AttachAddon";
import { FitAddon } from "@xterm/addon-fit";

import "@xterm/xterm/css/xterm.css";

// imports "Go"
import "./wasm_exec.js";

const term = new Terminal();

var attachAddon : null | AttachAddon = null;

const fitAddon = new FitAddon();
term.loadAddon(fitAddon);

// Polyfill for WebAssembly on Safari
if (!WebAssembly.instantiateStreaming) {
  WebAssembly.instantiateStreaming = async (resp, importObject) => {
    const source = await (await resp).arrayBuffer();
    return await WebAssembly.instantiate(source, importObject);
  };
}

function waitForDecode() {
  if(typeof decode !== "undefined"){
    startSession(urlData);
  } else {
    setTimeout(waitForDecode, 250);
  }
}

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

term.open(document.getElementById("terminal"));
fitAddon.fit();
window.onresize = () => {
  fitAddon.fit();
  if (attachAddon) {
    const dimensions = fitAddon.proposeDimensions();
    dimensions ? attachAddon.setSize(dimensions) : null;
  }
};
term.write("Welcome to the WebTTY web client.\n\r");

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

  attachAddon = new AttachAddon(sendChannel);
  term.loadAddon(attachAddon);

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
    waitForDecode();
    firstInput = true;
  } catch (err) {
    console.log(err);
  }
}

if (firstInput == false) {
  term.write("Run webtty and paste the offer message below:\n\r");
}

term.onData(data => {
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
