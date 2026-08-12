# isolation-probe banner

A single-file Node app (no dependencies) for the container-isolation talk. It
ships in two brandings, one per venue:

| File               | Theme                                | Vocabulary                         |
|--------------------|--------------------------------------|------------------------------------|
| `server.cyc.js`    | Commit Your Code (CYC26), blue       | `ESCAPED` / `LEAK` / `CONTAINED`    |
| `server.nagios.js` | [Nagios](https://www.nagios.org/), dark + orange | `CRITICAL` / `WARNING` / `OK` |

Both read the log written by `../scripts/evilnode.sh` (the fake `node` escape
probe) and show the same two states:

- **Vulnerabilities found** → a list of every boundary that would let an attacker
  escape or leak host info.
- **All boundaries CONTAINED** → `YOU'RE SECURE` (cyc) / `ALL SERVICES OK` (nagios).

They re-read the log on every request and the page auto-polls every 2.5s, so
during the talk you just re-run the probe and the browser updates itself — no
restart.

## Run

```sh
npm run start:cyc       # or: node server.cyc.js
npm run start:nagios    # or: node server.nagios.js
# open http://localhost:3000
```

In the container image, `Dockerfile.app` copies the selected theme to
`/app/server.js` (see the `THEME` build arg), so `CMD` stays theme-agnostic:

```sh
bash scripts/setup.sh cyc       # default
bash scripts/setup.sh nagios
# or: make setup-node THEME=nagios
```

## Config

| Env           | Default                                                           | Meaning                          |
|---------------|------------------------------------------------------------------|----------------------------------|
| `PORT`        | `3000`                                                            | HTTP port                        |
| `NODE_LOG`    | first of `/var/log/node.log`, `/tmp/node.log`                     | which probe log to read          |

Point `NODE_LOG` at the container's log (e.g. a bind-mounted path) if you run
the app on the host but the probe inside the container.

## Presentation flow

1. Run the container with **no** namespaces → run `node` (the probe) → refresh: lots of vulns.
2. Add each `CLONE_NEW*` flag, re-run `node`, refresh: rows disappear one by one.
3. Fully hardened → all boundaries green.

> The log is append-only; the app parses only the **most recent** run.
