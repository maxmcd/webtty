# WebRTTY

WebRTTY allows you to share a terminal session from your machine using WebRTC. You can share a server session with a different computer that also has WebRTTY installed, or you can open a WebRTTY session using this static page: https://maxmcd.github.io/webrtty/

WebRTTY uses a WebRTC DataChannel to send data between sources. The implementaiton is quite buggy at the moment, but it seems to work for the most part.

## Installation
```bash
go get -u github.com/maxmcd/webrtty
```
WebRTTY uses the wonderful [pions/webrtc](https://github.com/pions/webrtc) for WebRTC communication. It currently requires OpenSSL. More here: https://github.com/pions/webrtc#install

## Running

```bash
# On the host computer
$ webrtty
Setting up WebRTTY connection.

Connection ready. To connect to this session run:

webrtty eJysktFumzAUhu+ReIfzAjAbjI2PxEUIZEqVtemSTNvuXEwSpmAsY9rt7SfSrhfVNGlS5Rvb59f3HVvnsSBhMBQRSJqILBU8BZoxygRLmYD1Lay3DEh8XWEwFlEY+IIACQNVHDtzap11nfE4nlWUZBxIiTXHbIHJElcMU4n1CqsFSo6rFPMKS4FlilxgTnFR44ogEShLLAkKjkxgmeNqiQuBpMaMYEJRCKxzTCus6ll7csNksTzcVpsatPIqDPpCWXvpGuW7wYCEar/Zfdgt91vICCFh0BRvX6KKsfWTRdV4q8Zxvug7jc+4uWi0a5vH677xtlcWZxQ8tQ/ON9Gca87KmPYClCRsznVNG01Hp07YWbPZ6L4+fD7Y+laPf6r2SePd/ffKfNqsHw7fPur+x70qy/XNl5tx+3W573d3hznbKKM7rXyLk7avB6AwaQsJYTlQmcSU53HOYy4hSygl4H9ZOA+jh1NrWvf8FeQfuOSKS4UU78N7aS8XOYeExZLENJMxSyGTIpdX3OiOl5/glNbujdLZwfmX5H/1nwku3tnXGh0Nx+jVNP5lXH4HAAD//43T7sM=

When you have the answer, paste it below and hit enter:
```
From there you can pase the command to a different machine, or enter the base64 string into https://maxmcd.github.io/webrtty/
