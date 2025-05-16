# VRC-Stream-Haven

VRChat Stream Haven is a self-hosted CDN designed for publishing RTSP signals to the web. </br>
It allows users to join together into a Haven, improving stream stability for any viewer. </br>

It is currently still in very early development, pre-Alpha almost, so expect bugs and missing features. </br>

## Requirements

VRC-Stream-Haven uses the IP2Location LITE database for <a href="https://lite.ip2location.com">IP geolocation</a>. <br>

We use this location system to calculate the distance between the Flagship/Escorts and the client that wants to watch the stream. <br>
The location of each Escort gets saved in RAM and in logs but the location of each viewer is not stored long term.

This database is not provided by default in the repository due to licensing. <br>
VRC-Stream-Haven will work without this database but for optimisation it is recommended to use it, especially if you're planning on hosting a larger stream with many clients.

Simply create an account on <a href="https://lite.ip2location.com">IP geolocation</a>.<br>
Then go to https://lite.ip2location.com/database-download and look for your `Download Token`.<br>
Insert this download token into the `variables.txt` file that is next to your `.exe`.<br>
Make sure to insert it after `IP2LocationDownloadToken=` <br>
It should look something like this `IPLocationDownloadToken=XXXXXXXXXXXXX`

That's it!
Each time you now start VRC-Stream-Haven it will check if there is a newer version of the database and download it.

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

- [x] PoC
- [ ] Refactor code base to make it more readable.
