# IBC Relayer Docker Context

This directory serves as the Docker build context for the IBC relayer container. It's required by Tilt for the Docker build process but doesn't need to contain any files since the Dockerfile is self-contained.

The Dockerfile (`../dockerfiles/ibc-relayer.dockerfile`) automatically detects the target architecture and downloads the appropriate Hermes binary for both x86_64 and ARM64 systems.
