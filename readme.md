# WebRTTY

WebRTTY allows you to share a terminal session from your machine using WebRTC. You can pair with a friend without setting up a proxy server, debug servers behind NATs, and more. WebRTTY also works in-browser. You can connect to a WebRTTY session from this static page:  https://maxmcd.github.io/webrtty/

### Status

There are a handful of bugs to fix, but everything works pretty well at the moment. Please open an issue if you find a bug. 

### Installation
```bash
go get -u github.com/maxmcd/webrtty
```
WebRTTY uses the wonderful [pions/webrtc](https://github.com/pions/webrtc) for WebRTC communication. It currently requires OpenSSL to build. More here: https://github.com/pions/webrtc#install

### Running

```
$ webrtty -h 
Usage of ./webrtty:
  -cmd
        The command to run. Default is "bash -l"
        Because this flag consumes the remainder of the command line,
        all other args (if present) must appear before this flag.
        eg: webrtty -o -v -ni -cmd docker run -it --rm alpine:latest sh
  -ni
        Set host to non-interactive
  -non-interactive
        Set host to non-interactive
  -o    One-way connection with no response needed.
  -v    Verbose logging
```

```
# On the host computer
$ webrtty
Setting up a WebRTTY connection.

Connection ready. Here is your connection data:

25FrtDEjh7yuGdWMk7R9PhzPmphst7FdsotL11iXa4r9xyTMR34koAauQYivKViWYBskf8habEc5vHf3DZge5VivuAT79uSCvzc6aLXb2M11kcUn9rzb4DX4KSaKB5PhszAiiCB2iHKugPUwhzCMd7JjYU2ZLWGHBnjCf1cujfMx4E1ZdZADDh1FaQ2njy6Chnmxhy68hKZh8HX3SqQKShEffyvptyun69c3cMBXBwq4eHZdrX86KMaQLFSCZaoPh8sLaRydPqiyHw9tYmAZ6GAFUQoju72SSPmT8sV3T4of4ZqCWm91EJfazmqVst8D9sxqj5HS9kYuBcn78Pa1gFY85hTehmHabQXh8XXouHyHp84yfWJqEQTAn7J8sGkH7SR4p8c24ohg2yfWbHSWzBD7Bv5Zz7MXmcDSzXMFpSzgqEtNKKokhqFmCHd6UXGEW4dCpAB37ANzBqsM4tn1YAfNny

Paste it in the terminal after the webrtty command
Or in a browser: https://maxmcd.github.io/webrtty/

When you have the answer, paste it below and hit enter:
```

### Terminal Size

Other terminal sharing tools have various fixes to deal with different terminal sizes from different users. WebRTTY just forces the use of the client session terminal size, so the host side of things can frequently render incorrectly. I think the easiest way to fix this would be to leverage `tmux` sessions and simply provide an option to proxy a shared `tmux` session instead of `bash`. Might add that at a later date. 

### One-way Connections

One-way connections can be enabled with the `-o` flag. A typical webrtc connection requires an SDP exchange between both parties. By default, WebRTTY will create an SDP offer and wait for you to enter the SDP answer. With the `-o` flag the initial offer is sent along with a public url that the receiver is expected to post their response to. This uses my service [10kb.site](https://www.10kb.site). The host then polls the url continually until it gets an answer.

I think this somewhat violates the spirit of this tool because it relies on me to not snoop on connection details. However, one-way connections allow you to do very cool things. Eg: I can have a build server output a WebRTTY connection string on error and allow anyone to attach to the session.

Maybe just encrypt the SDP and only allow the `-o` flag with a shared secret?

Very open to any ideas on how to enable trusted one-way connections. Please open an issue or reach out if you have thoughts. For now, the `-o` flag will print a warning and link to this explanation. 

