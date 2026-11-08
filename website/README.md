# website: ephesus.church

This folder is the program that serves this repository as https://ephesus.church. Every Markdown file is
rendered to a page (a folder's `README.md` is that folder's page: `/`, `/usdollar/`, `/revelation12/`), and every
other file (the images, the PDFs, the scripts in `revelation12/tools/`) is served as it is, each at the path it
has here, so relative links and `<img src="./images/...">` tags work as they do on GitHub. A folder without a
`README.md` gets a plain list of its files. Nothing whose path starts with a dot (`.git`) is served. The sidebar
holds the page list (`pages` in `main.go`) and the page's table of contents.

Who wrote what: [AUTHORSHIP.md](AUTHORSHIP.md).

## Run it

    cd website && go run .                 # serves the checkout it sits in, on http://127.0.0.1:5001
    ephesus -content DIR -listen ADDR      # DIR is a checkout of this repository

The template and the stylesheet are built into the binary; the pages are read from the checkout on every
request, so an edit shows on the next reload.

## How it is deployed

On the kourt box (138.68.5.120, hostname `mysites`):

- `/usr/local/bin/ephesus`, run by `ephesus.service` ([deploy/ephesus.service](deploy/ephesus.service)) as a
  throwaway user that can write nothing, on 127.0.0.1:5001. The box's nginx site `ephesus_church` (certbot's
  certificate) sends https://ephesus.church there, and HTTP redirects to HTTPS. The box's ModSecurity checks the
  requests but, for this site only, not the replies (`modsecurity_rules 'SecResponseBodyAccess Off'`): its
  data-leakage rules cut the scripts in `revelation12/tools/` off mid-reply.
- `/srv/ephesus`, a checkout of this repository from GitHub. `ephesus-update.timer` runs
  `/usr/local/sbin/ephesus-update` ([deploy/ephesus-update](deploy/ephesus-update)) every ten minutes, which
  brings it to GitHub's `main`, so a push is on the site within about ten minutes, with no restart. To have it at
  once: `ssh root@138.68.5.120 systemctl start ephesus-update`.
- A change to this program needs a new build. On the devbox, `make deploy` here runs gofmt, vet, the tests and
  the static build (go.mod asks for Go 1.27.3, which the go command fetches), then
  [deploy/install.sh](deploy/install.sh) `root@138.68.5.120`, which copies the binary, the units and the update
  script, clones `/srv/ephesus` if it is missing, restarts the service and checks that it answers.

Logs: `journalctl -u ephesus` and `journalctl -u ephesus-update`.
