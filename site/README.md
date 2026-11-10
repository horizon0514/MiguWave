# MiguWave website

Static landing page for [MiguWave](https://github.com/horizon0514/MiguWave), served at
`https://music.colaorange.me/`.

## Files

- `index.html` — page structure and copy
- `styles.css` — layout, responsive breakpoints and motion
- `main.js` — mobile navigation, reveal animations and interface tabs
- `assets/` — optimized WebP screenshots from the desktop app
- `og.<hash>.png` — content-hashed social preview image
- `og-source.html` — 1200 × 630 social card source (`noindex`)
- `_headers` — Cloudflare Pages security and caching headers
- `robots.txt`, `sitemap.xml` — search engine metadata

No build step, fonts, analytics or third-party scripts are required.

## Local preview

```bash
python3 -m http.server 4173 --directory site
```

Open `http://localhost:4173/`.

## Deploy to Cloudflare Pages

The production site is a Cloudflare Pages project named `miguwave`. Deploy the
contents of this directory, not the repository root.

To let Cloudflare deploy from GitHub, connect the repository to the Pages
project and use:

- Production branch: `main`
- Root directory: `site`
- Build command: empty
- Build output directory: `/`

The current Direct Upload project can also be updated by uploading this
directory from the Cloudflare dashboard. No bundler, font, analytics or
third-party runtime is required.

Custom domain: `music.colaorange.me`. Keep it pointed at the `miguwave` Pages
project; changing the deployment source does not change the public URL.

## Download links

Download buttons always point to the latest release through stable asset names:

```text
https://github.com/horizon0514/MiguWave/releases/latest/download/miguwave.dmg
https://github.com/horizon0514/MiguWave/releases/latest/download/miguwave-setup-amd64.exe
https://github.com/horizon0514/MiguWave/releases/latest/download/miguwave-setup-arm64.exe
```

The release workflow publishes both versioned files and these stable aliases, so the
site does not need to be updated for each release.

## Regenerating the social card

Open `og-source.html` at a 1200 × 630 viewport and capture the page as a
content-hashed `og.<hash>.png`, then update the social image meta tags.
