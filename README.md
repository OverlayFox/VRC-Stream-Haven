# VRC-Stream-Haven
VRChat Stream Haven is a locally hosted Docker application designed for publishing RTMP or SRT signals to the web as a RTSP signal. </br>
The main focus of VRC Stream Haven is privacy of the people who are pushing the stream.

It also has the ability for users to join as nodes, improving stream stability for any viewer. </br>

Primarily built for VRChat, it's versatile enough for other applications, providing a practical solution for content streaming and collaboration.

## Q&A
### Why stick with RTSP and not HLS?
VRChat requires any http protocol to be encrypted, so the server serving the files, needs to have a valid certificate attached to it. </br>
To get a valid certificate a server needs a domain. This domain needs to be purchased with a name, address and payment method. </br>
This makes anything the server serves traceable to the owner of the domain. </br>
This is why we chose RTSP. </br>
RTSP is an RTP protocol that focuses on multi-client single-sever setups.
Unlike RTMP or SRT which focus on single-client single-server setups. </br>

RTSP isn't an encrypted protocol so your stream will be accessible to anyone on the web. </br>
This is why the system automatically generates a password for your stream and a unique stream key. </br>
Viewers will still be able to easily access the stream in VRChat without needing to do anything but network sniffers won't easily be able to access them which adds a little bit of protection to the outgoing signal.

### How does the node system work?
A friend can decide to join the "Haven" for your stream setup by sharing their public-key with you that is automatically generated for them. </br>
Once you added their public key and region to the config file they can join your VPN network. </br>
Your server will then push one RTMP stream to that VPN network and any server that joins the VPN can then pull the stream and re-stream it as an RTSP stream from their server. </br>
Due to the VPN the entire stream between you and the node is encrypted and not accessible to the public internet. </br>
Only the RTSP stream will be accessible to the viewers.

If a viewer now requests the stream from your server a system will check which node is closest to that viewer and will redirect it to them instead.</br>
Your server will stay a node in this mode.

### What happens if a node suddenly goes offline?
If a node goes offline the viewers will re-request the stream from the main server. </br>
Before rerouting the viewer to a node the main server will ping the node to see if it's online. If it isn't another node close to the viewer will be picked.

### Can I add a node while the stream is live?
Yes, you can add nodes while you're already streaming. The node will only start being used by new joining viewers. Already watching viewers need to reconnect to be rerouted to the node.