# VRC-Stream-Haven

VRChat Stream Haven is a locally hosted CDN designed for publishing RTSP signals to the web. </br>
It allows users to join together as nodes, improving stream stability for any viewer. </br>

Primarily built for VRChat, it's versatile enough for other applications, providing a practical solution for content
streaming and collaboration.

## Q&A

### Why was this project made?

VRChat requires any HLS stream to be encrypted via TLS. </br>
This makes it difficult to stream to VRChat without a valid domain and certificate. </br>

So a lot of people use RTSP to stream into VRChat because it doesn't require encryption from VRChats site. </br>
But RTSP often runs into package timeout issue if the distance between the server and client are too long. </br>

The solution is to make the distance shorter. </br>
This can be done with a CDN. </br>
But CDNs are expensive, and often require a lot of setup. </br>

So this project enables friends to join together as one CDN network, re-streaming the servers video signal to the web
from their location. </br>

### How does it work?

The server streams the RTSP signal to the web. </br>
Once a client tries to establish a connection with the server, the server will check if a node is closer to the client
then the server is. </br>
If a node is closer, the server will redirect the client to the node. </br>

### How are the nodes connected to the server?

The server setups a Wireguard VPN. </br>
The nodes then join the VPN and start reading the HLS signal that is being provided by the server. </br>
Each node then converts that HLS signal back to RTSP and streams it to the web. </br>