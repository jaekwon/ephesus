# Authorship: website/

The writing in this repository is Jae Kwon's, and so is the site. The renderer in this folder was committed by
him on 2026-03-09 (b64a380, 49f8364, 9adab44, b6e4fd2): server-side Markdown rendering, the sticky table of
contents with scroll tracking, the page list, the image thumbnails that open in a lightbox, and the redirects from
`README.md` paths to their pages. Its look is unchanged.

On 2026-11-08 Claude (Anthropic's model, run through Claude Code) put it back online at his request. The
`Co-Authored-By: Claude ...` trailer on that commit records which model typed it; it does not mean the site or
its design were Claude's.

## Jae's prompt, verbatim (2026-11-08)

> maybe make a subagent figure out how to get ephesus.church to render too... it used to point to a digital ocean
> node or something.

What this set: ephesus.church shows this repository again, from his own server.

## His standing rules that shaped the work

- Nothing of his on Anthropic's hosting: the site runs on his kourt box, and nothing went to claude.ai.
- No heavy work on the Mac: it was built and tested on the devbox.
- Dependencies must be annealed: gomarkdown stays at the April 2026 commit that fixes the one advisory the Go
  vulnerability database lists against the version it had (GO-2026-5208), not at the much-rewritten November
  head; the build uses Go 1.27.3, the patched toolchain. govulncheck finds nothing.

## What Claude decided on its own

- Serving every file at its path in the repository, folders with a trailing slash, in place of three hard-coded
  pages and image folders; the old `/usdollar` and `/jesus_and_taxes` addresses redirect to `/usdollar/` and
  `/jesus_and_taxes/`.
- The `-content` and `-listen` flags (it listens on the loopback by default, where it used to listen on every
  address); the template and stylesheet built into the binary; plain file lists for folders without a README.md;
  no dot paths; saved web pages served sandboxed.
- "Revelation 12" in the sidebar's page list, and the page list shown on pages without headings too.
- The deployment: `/srv/ephesus` as a checkout of GitHub's main, pulled every ten minutes, and the hardened
  systemd units in `deploy/`.
