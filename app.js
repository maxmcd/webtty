Terminal.applyAddon(attach);
Terminal.applyAddon(fullscreen);
Terminal.applyAddon(fit);

var term = new Terminal();
term.open(document.getElementById("terminal"));
term.toggleFullScreen();
term.fit();
window.onresize = () => {
  term.fit();
};
term.write("Welcome to the WebRTTY web client.\n\r");
term.write("Run webrtty and paste the offer message below:\n\r");
let firstInput = false;
term.on("data", data => {
  if (!firstInput) {
    term.reset();
    try {
      startSession(data);
    } catch (err) {
      term.write(`There was an error with the offer: ${data}\n\r`);
      term.write("Try entering the message again: ");
      return;
    }
    firstInput = true;
  }
});

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
  sendChannel.send(JSON.stringify(["stdin", term.rows, term.cols]));
  console.log("sendChannel has opened");
};
// sendChannel.onmessage = e => {}

pc.onsignalingstatechange = e => log(pc.signalingState);
pc.oniceconnectionstatechange = e => log(pc.iceConnectionState);
pc.onicecandidate = event => {
  if (event.candidate === null) {
    term.write(
      "Answer created. Send the following answer to the host:\n\r\n\r"
    );
    term.write(encodeOffer(pc.localDescription.sdp));
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

startSession = data => {
  pc
    .setRemoteDescription(
      new RTCSessionDescription({
        type: "offer",
        sdp: decodeOffer(data)
      })
    )
    .catch(log);
  pc
    .createAnswer()
    .then(d => pc.setLocalDescription(d))
    .catch(log);
};

encodeOffer = data => (
  return btoa(
    pako.deflate(JSON.stringify({ Sdp: data }), {
      to: "string"
    })
  );
);

decodeOffer = data => (
  JSON.parse(pako.inflate(atob(data), { to: "string" })).Sdp;
);
