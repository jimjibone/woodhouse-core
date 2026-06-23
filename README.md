# woodhouse-core - woodhouse core server

This repository contains the Woodhouse core server.

## History

This repository was the original home to woodhouse-api, wh, and numerous bridges.
Before making the project public the mono-repo was split into multiple repos to:

- Make it easier to apply different licensing (i.e. `woodhouse-core` is AGPLv3
  to protect the open source roots of the core project, `woodhouse-api` and `wh`
  are Apache v2 to allow easier integration into other projects).
- Make it easier to import the API and wh module into other projects.
- Allow client development to be independent from the core and other clients.
