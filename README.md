
# broadcastd
RTMP re-stream daemon

## Quick Start Guide

1. Add your account info in `config.yaml`.
2. Build and run the containers:
```
docker-compose up --build
```
3. Using [OBS]([https://obsproject.com/](https://obsproject.com/)) or other compatible software, configure the output to:
- Server: `rtmp://localhost/live`
- Stream Key: `test`
4. Start streaming and it should appear live on all the accounts.

## TODOs
- Handle 2FA login.
- Add an option to provide own IGTV thumbnail.
- Post a pinned comment near the 60-min mark to indicate the live will continue after a restart.
- Publish viewer count metrics to Prometheus endpoint.

## Pull Requests
Yes please.

## Acknowledgements
Parts of the code are based on the following projects:
- https://github.com/dilame/instagram-private-api
- https://github.com/ahmdrz/goinsta
