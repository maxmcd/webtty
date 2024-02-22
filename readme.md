![](https://raw.githubusercontent.com/maxmcd/webtty/70f7911f4e69dffe3eb3cfd6ad9dd8060dc10dd5/out.gif)

## WebTTY

WebTTY allows you to share a terminal session from your machine using WebRTC. You can pair with a friend without setting up a proxy server, debug servers behind NATs, and more. WebTTY also works in-browser. You can connect to a WebTTY session from this static page:  [https://maxmcd.github.io/webtty/](https://maxmcd.github.io/webtty/)

### Status

There are a handful of bugs to fix, but everything works pretty well at the moment. Please open an issue if you find a bug.

### Installation

Download a binary from the releases page: https://github.com/maxmcd/webtty/releases

Or, install directly with Go. WebTTY requires go version 1.9 or higher.

```bash
go install github.com/maxmcd/webtty@latest
```

There were recent breaking api changes in the pion/webrtc library. Make sure to run `go get -u github.com/pion/webrtc` if you're running into any installation errors.

### Running

```shell
> webtty -h
Usage of webtty:
  -cmd
        The command to run. Default is "bash -l"
        Because this flag consumes the remainder of the command line,
        all other args (if present) must appear before this flag.
        eg: webtty -o -v -ni -cmd docker run -it --rm alpine:latest sh
  -ni
        Set host to non-interactive
  -non-interactive
        Set host to non-interactive
  -o    One-way connection with no response needed.
  -s string
        The stun server to use (default "stun:stun.l.google.com:19302")
  -v    Verbose logging
```

#### On the host computer

```shell
> webtty
Setting up a WebTTY connection.

Connection ready. Here is your connection data:

25FrtDEjh7yuGdWMk7R9PhzPmphst7FdsotL11iXa4r9xyTM4koAauQYivKViWYBskf8habEc5vHf3DZge5VivuAT79uSCvzc6aL2M11kcUn9rzb4DX4...

Paste it in the terminal after the webtty command
Or in a browser: https://maxmcd.github.io/webtty/

When you have the answer, paste it below and hit enter.
```

#### On the client computer

```shell
> webtty 25FrtDEjh7yuGdWMk7R9PhzPmphst7FdsotL11iXa4r9xyTM4koAauQYivKViWYBskf8habEc5vHf3DZge5VivuAT79uSCvzc6aL2M11kcUn9rzb4DX4...

```

### Terminal Size

By default WebTTY forces the size of the client terminal. This means the host size can frequently render incorrectly. One way you can fix this is by using tmux:

```bash
tmux new-session -s shared
# in another terminal
webtty -ni -cmd tmux attach-session -t shared
```
Tmux will now resize the session to the smallest terminal viewport.

### One-way Connections

One-way connections can be enabled with the `-o` flag. A typical webrtc connection requires an SDP exchange between both parties. By default, WebTTY will create an SDP offer and wait for you to enter the SDP answer. With the `-o` flag the initial offer is sent along with a public url that the receiver is expected to post their response to. This uses my service [10kb.site](https://www.10kb.site). The host then polls the url continually until it gets an answer.

I think this somewhat violates the spirit of this tool because it relies on a third party service. However, one-way connections allow you to do very cool things. Eg: I can have a build server output a WebTTY connection string on error and allow anyone to attach to the session.

SDP descriptions are encrypted when uploaded and encryption keys are shared with the connection data to decrypt. So presumably the service being compromised is not problematic.

Very open to any ideas on how to enable trusted one-way connections. Please open an issue or reach out if you have thoughts. For now, the `-o` flag will print a warning and link to this explanation.
