encodeOffer = data =>
  btoa(
    pako.deflate(JSON.stringify({ Sdp: data }), {
      to: "string"
    })
  );

decodeOffer = data => JSON.parse(pako.inflate(atob(data), { to: "string" }));

create10kbFile = (path, body) =>
  fetch("https://up.10kb.site/" + path, {
    method: "POST",
    body: body
  })
    .then(resp => resp.text())
    .then(resp => {});

startSession = data => {
  sessionDesc = decodeOffer(data);
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

Terminal.applyAddon(attach);
Terminal.applyAddon(fullscreen);
Terminal.applyAddon(fit);
var TenKbSiteLoc = null;

var term = new Terminal();
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
  sendChannel.send(JSON.stringify(["stdin", term.rows, term.cols]));
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


let firstInput = false;
urlData = window.location.hash.substr(1);
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
      term.write(`There was an error with the offer: ${data}\n\r`);
      term.write("Try entering the message again: ");
      return;
    }
    firstInput = true;
  }
});
