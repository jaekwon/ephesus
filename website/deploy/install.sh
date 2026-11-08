#!/bin/sh
# install.sh BOX: put ephesus.church on BOX (the kourt box, root@138.68.5.120): the binary as /usr/local/bin/ephesus
# and its systemd unit, the content checkout at /srv/ephesus (cloned from GitHub the first time), and
# ephesus-update with its timer, which keeps that checkout at GitHub's main. Restarts the service and checks that it
# answers. `make deploy` runs the gate first. The box's nginx site ephesus_church (certbot's) already sends
# ephesus.church to 127.0.0.1:5001; this script leaves nginx alone.
set -eu
box=${1:?usage: deploy/install.sh root@host}
cd "$(dirname "$0")/.."
scp -q bin/ephesus "$box:/usr/local/bin/ephesus.new"
scp -q deploy/ephesus-update "$box:/usr/local/sbin/ephesus-update.new"
scp -q deploy/ephesus.service deploy/ephesus-update.service deploy/ephesus-update.timer "$box:/etc/systemd/system/"
ssh "$box" 'set -e
chmod 755 /usr/local/bin/ephesus.new /usr/local/sbin/ephesus-update.new
mv /usr/local/bin/ephesus.new /usr/local/bin/ephesus
mv /usr/local/sbin/ephesus-update.new /usr/local/sbin/ephesus-update
if [ ! -d /srv/ephesus/.git ]; then
  umask 022
  git clone -q https://github.com/jaekwon/ephesus.git /srv/ephesus
fi
systemctl daemon-reload
systemctl start ephesus-update.service
systemctl enable -q ephesus
systemctl restart ephesus
systemctl enable -q --now ephesus-update.timer
ok=
for i in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS -o /dev/null http://127.0.0.1:5001/; then ok=1; break; fi
  sleep 1
done
[ -n "$ok" ] || { journalctl -u ephesus -n 20 --no-pager; echo "ephesus does not answer" >&2; exit 1; }
echo "ephesus answers on 127.0.0.1:5001, serving /srv/ephesus at $(git -C /srv/ephesus log -1 --format="%h %s")"'
