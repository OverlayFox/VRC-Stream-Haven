# VRC-Stream-Haven
**V**igorously **R**esentful **C**ontent Stream Haven is a locally hosted application designed for publishing RTMP or SRT signals to the web as a RTSP or HLS streams. </br>

VRCHavens main strength lies in the node network feature.<br>
Users can choose to join together as nodes, to form a private CDN for the stream, improving stability for viewers. </br>

## General Functionality
The Streamer publishes their stream to the Server via SRT or RTMP. <br>
The server then re-streams that signal to all of the nodes over a SRT connection.<br>
The server and nodes then convert the SRT signal to RTSP and push it out to the web.<br>

The viewer will connect to the server and ask for the stream.<br>
The server will look up the country from where the viewer is and see which node is the closest to the viewer.<br>
It will then redirect the viewer to the closest node.<br>

## Signal Flow
```
                        ┌────┐       RTSP        ┌──────┐ 
                        │Node├──────────────────►│Viewer│ 
                        └────┘                   └──────┘ 
                          ▲                               
                          │                               
                          │SRT                            
                          │                               
┌────────┐  SRT/RTMP   ┌──┴───┐      RTSP         ┌──────┐
│Streamer├────────────►│Server├──────────────────►│Viewer│
└────────┘             └──┬───┘                   └──────┘
                          │                               
                          │SRT                            
                          │                               
                          ▼                               
                        ┌────┐       RTSP         ┌──────┐
                        │Node├───────────────────►│Viewer│
                        └────┘                    └──────┘
```

## But....why?
VRCHaven is a general streaming tool but has been optimized to work best with the video game [VRChat](https://hello.vrchat.com/)<br>
When streaming into VRChat worlds, viewers that are very far away from the source, might not be able to see the stream or will see it with a lot of artefacting.<br>
This is mainly due to pacakge timeouts, which are especially apparent when using lower latency protocols like RTSP.

This can be fixed by using a CDN (Content Delivery Network).<br>
But those can be expensive, difficult to setup, introduce a lot of latency and in certain countries might not allow the streaming of certain content.<br>

This is where VRCHaven comes in, it allows you to setup a private CDN that your friends can join in on to shorten the distance between the source and the viewer.

## Security information - IMPORTANT PLEASE READ THIS!
VRCHaven does not encrypt the outgoing HLS or RTSP stream or obscure your IP Address.<br>
This means your IP address and the IP address of all nodes will be exposed to anyone watching the stream.<br>
Which can be used to figure out where the server and nodes are. Which might be your home address.<br>

This is why we recommend that the server and each node routes the RTSP or HLS output of VRCHaven over a VPN or proxy provider that you trust.
It only needs to support port forwarding.
This will obscure the IP address of the nodes and server.
