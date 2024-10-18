# VRC-Stream-Haven

VRChat Stream Haven is a self-hosted CDN designed for publishing RTSP signals to the web. </br>
It allows users to join together into a Haven, improving stream stability for any viewer. </br>

It is currently still in very early development, pre-Alpha almost, so expect bugs and missing features. </br>

## How it works:

One server acts as the main server (Flagship). </br>
The Flagship will receive the main SRT feed that will be sent to all viewers. </br>

Another server can act as a Node (Escort). </br>
The Escort will call the Flagship via an API and request to join the Haven. </br>
If the passphrase is correct, the Flagship will add the Escort to the Haven. </br>
This allows the Escort to pull the SRT feed from the Flagship and then remux it to RTSP. </br>

When a viewer connects to the Flagship, the Flagship will locate the viewer's IP and send them to the Escort that is
closest to the viewer. </br>

## Roadmap:

- [x] Implement an API that will let the Escort and Flagship communicate with each other.
- [x] Add simple content encryption to the API calls.
- [x] Let each Escort define how many viewers they can handle.
- [ ] Refactor code base to make it more readable.
- [ ] Add unittests.
- [ ] Add leech mode, where a viewer can join the Haven while keeping their RTSP traffic in their LAN.
- [ ] Implement a FFMPEG daemon that will transcode the SRT signal to RTSP on each escort for less overhead.
- [ ] Remove MediaMTX and use a self build SRT Server.
- [ ] Build a release pipeline.
- [ ] Write a How-To guide for Linux and Windows.
- [ ] Realtime Metrics about each Escort and about the Flagship.
- [ ] Add a UI for the Flagship and for each Escort.
