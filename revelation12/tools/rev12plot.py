#!/usr/bin/env python3
"""rev12plot.py OUT.png [DATE ...]: draw the Leo-Virgo sky on the given dates (default 1293-09-12, 1814-09-17,
2017-09-23, the three days tools/rev12.py found) so the Revelation 12 configurations can be compared by eye.

Each panel is the same patch of sky (north up, east left, J2000): the stick figures and named stars of Leo and
Virgo, the Sun at Virgo (clothing her), the Moon (drawn four times its real size so it shows), Jupiter with its path
over the previous 400 days (the "child", and its retrograde loop), and Mercury, Venus and Mars in Leo (with Leo's
nine named stars, the crown of twelve). Positions from astropy's built-in ephemeris, geocentric, arcminute-level.

Needs astropy and matplotlib.
"""
import sys
import warnings

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np
from astropy.coordinates import SkyCoord, get_body, solar_system_ephemeris
from astropy.time import Time

warnings.filterwarnings("ignore")
solar_system_ephemeris.set("builtin")

args = sys.argv[1:]
eph_dir = None
if "--eph" in args:
    i = args.index("--eph")
    eph_dir = args[i + 1]
    del args[i:i + 2]
out = args[0] if args else "rev12.png"
dates = args[1:] or ["1293-09-12", "1814-09-17", "2017-09-23"]

if eph_dir:
    # JPL DE441 through skyfield, for dates astropy's built-in ephemeris cannot reach (before 1000 BC, after 3000)
    from skyfield.api import load as sf_load
    _ts = sf_load.timescale()
    _parts = {}
    def _eph(jd):
        k = 1 if jd < 2440400.5 else 2
        if k not in _parts:
            _parts[k] = sf_load(f"{eph_dir}/de441_part-{k}.bsp")
        return _parts[k]
    _names = {"sun": "sun", "moon": "moon", "mercury": "mercury", "venus": "venus", "mars": "mars barycenter", "jupiter": "jupiter barycenter"}
    def get_body(name, t):
        jd = np.atleast_1d(t.tt.jd)
        e = _eph(float(jd[0]))
        ra, dec, _ = e["earth"].at(_ts.tt_jd(jd)).observe(e[_names[name]]).radec()
        c = SkyCoord(ra=ra.hours * 15, dec=dec.degrees, unit="deg", frame="icrs")
        return c if np.ndim(t.tt.jd) else c[0]
    def parse(d):  # "+07889-12-07" or "-01850-07-31" (proleptic Gregorian, TT) or an ordinary ISO date
        y, m, dd = int(d[:-6]), int(d[-5:-3]), int(d[-2:])
        return Time(_ts.tt(y, m, dd, 12).tt, format="jd", scale="tt")
else:
    def parse(d):
        return Time(d + "T12:00:00", scale="utc")

# J2000 positions, degrees (Hipparcos). Leo's nine named stars first.
LEO = {"Regulus": (152.093, 11.967), "Denebola": (177.265, 14.572), "Algieba": (154.993, 19.842), "Zosma": (168.527, 20.524),
       "Chertan": (168.560, 15.430), "Adhafera": (154.173, 23.417), "Ras Elased": (146.463, 23.774), "Rasalas": (148.191, 26.007),
       "eta Leo": (151.833, 16.763)}
LEO_LINES = [("Ras Elased", "Rasalas"), ("Rasalas", "Adhafera"), ("Adhafera", "Algieba"), ("Algieba", "eta Leo"), ("eta Leo", "Regulus"),
             ("Regulus", "Chertan"), ("Chertan", "Denebola"), ("Denebola", "Zosma"), ("Zosma", "Algieba")]
VIR = {"Spica": (201.298, -11.161), "Zavijava": (177.674, 1.765), "Porrima": (190.415, -1.449), "Auva": (193.901, 3.398),
       "Vindemiatrix": (195.544, 10.959), "Heze": (203.673, -0.596), "Zaniah": (184.977, -0.667), "theta Vir": (197.488, -5.539),
       "Syrma": (214.004, -6.001), "mu Vir": (220.765, -5.658), "Khambalia": (214.778, -13.371), "tau Vir": (210.411, 1.545),
       "109 Vir": (221.563, 1.893), "nu Vir": (176.465, 6.529)}
VIR_LINES = [("nu Vir", "Zavijava"), ("Zavijava", "Zaniah"), ("Zaniah", "Porrima"), ("Porrima", "Auva"), ("Auva", "Vindemiatrix"),
             ("Porrima", "theta Vir"), ("theta Vir", "Spica"), ("Spica", "Heze"), ("Heze", "tau Vir"), ("tau Vir", "109 Vir"),
             ("Heze", "Syrma"), ("Syrma", "mu Vir"), ("Syrma", "Khambalia")]
FEET = SkyCoord(ra=[220.765, 214.004, 214.778], dec=[-5.658, -6.001, -13.371], unit="deg")

BG, INK, GOLD, DIM = "#0e1014", "#ecebe6", "#d6ad4a", "#6b707a"
ncol = min(len(dates), 4)
nrow = (len(dates) + ncol - 1) // ncol
fig, axes = plt.subplots(nrow, ncol, figsize=(7.2 * ncol, 7.6 * nrow), facecolor=BG, squeeze=False)
axes = [a for row in axes for a in row]
for a in axes[len(dates):]:
    a.set_visible(False)

for ax, d in zip(axes, dates):
    t = parse(d)
    ax.set_facecolor(BG)
    for name, (ra, dec) in {**LEO, **VIR}.items():
        ax.plot(ra, dec, "o", ms=3.2, color=INK, zorder=3)
        ax.annotate(name, (ra, dec), xytext=(4, 4), textcoords="offset points", color=DIM, fontsize=7)
    for stars, lines in ((LEO, LEO_LINES), (VIR, VIR_LINES)):
        for a, b in lines:
            ax.plot([stars[a][0], stars[b][0]], [stars[a][1], stars[b][1]], color=DIM, lw=0.8, alpha=0.7, zorder=2)

    # Jupiter's path over the previous 400 days, a dot every 50
    tt = t - np.arange(400, -1, -5) * 1.0
    jp = get_body("jupiter", tt)
    ax.plot(jp.ra.deg, jp.dec.deg, color=GOLD, lw=1.0, alpha=0.8, zorder=4)
    ax.plot(jp.ra.deg[::10], jp.dec.deg[::10], ".", color=GOLD, ms=3, alpha=0.8, zorder=4)

    bodies = {n: get_body(n, t) for n in ("sun", "moon", "jupiter", "mercury", "venus", "mars")}
    s = bodies["sun"]
    for r, a in ((6, 0.08), (4, 0.14), (2.2, 0.35)):
        ax.add_patch(plt.Circle((s.ra.deg, s.dec.deg), r, color="#ffd27a", alpha=a, zorder=5))
    ax.add_patch(plt.Circle((s.ra.deg, s.dec.deg), 1.2, color="#ffe9a8", zorder=6))
    ax.annotate("Sun", (s.ra.deg, s.dec.deg), xytext=(8, -14), textcoords="offset points", color="#ffe9a8", fontsize=9)
    m = bodies["moon"]
    ax.add_patch(plt.Circle((m.ra.deg, m.dec.deg), 1.0, color="#cfd3da", zorder=6))
    ax.annotate("Moon", (m.ra.deg, m.dec.deg), xytext=(8, -12), textcoords="offset points", color="#cfd3da", fontsize=9)
    for n, col, ms in (("jupiter", GOLD, 11), ("mars", "#e06a60", 7), ("venus", "#fff4d6", 8), ("mercury", "#b9a58a", 6)):
        b = bodies[n]
        ax.plot(b.ra.deg, b.dec.deg, "o", color=col, ms=ms, zorder=7, markeredgecolor=BG)
        ax.annotate(n.capitalize(), (b.ra.deg, b.dec.deg), xytext=(7, 7), textcoords="offset points", color=col, fontsize=9)

    # the numbers
    ps = [bodies[n] for n in ("mercury", "venus", "mars")]
    spread = max(ps[i].separation(ps[j]).deg for i in range(3) for j in range(i + 1, 3))
    feet = min(m.separation(f).deg for f in FEET)
    # days Jupiter had been in Virgo (IAU) before the date
    from astropy.coordinates import get_constellation
    jd = 0
    while jd < 700 and get_constellation(get_body("jupiter", t - (jd + 5) * 1.0), short_name=True) == "Vir":
        jd += 5
    ax.set_title(f"{d}\nMoon {feet:.1f}° from Virgo's feet · Jupiter {jd} days in Virgo · crown {spread:.1f}° wide",
                 color=INK, fontsize=11, pad=12)

    ax.set_xlim(232, 138)  # east on the left, as on the sky
    ax.set_ylim(-26, 34)
    ax.set_aspect("equal")
    ax.set_xlabel("right ascension (°)", color=DIM)
    ax.set_ylabel("declination (°)", color=DIM)
    ax.tick_params(colors=DIM)
    for sp in ax.spines.values():
        sp.set_color(DIM)
    ax.grid(color=DIM, alpha=0.15, lw=0.5)

fig.suptitle("The Revelation 12 sky: Sun in Virgo, Moon at her feet, Jupiter long in Virgo, Mercury, Venus and Mars in Leo"
             " (with Leo's nine named stars: twelve). Gold line: Jupiter's path over the previous 400 days.",
             color=INK, fontsize=12, y=0.98)
fig.text(0.5, 0.012, ("Geocentric, J2000, " + ("JPL DE441 via skyfield; dates in Terrestrial Time, proleptic Gregorian" if eph_dir else "astropy built-in ephemeris (arcminute-level)") +
         ". The Moon is drawn four times its real size. tools/rev12plot.py, denaro.org"),
         color=DIM, fontsize=8, ha="center")
fig.tight_layout(rect=(0, 0.03, 1, 0.95 if nrow == 1 else 0.965))
fig.savefig(out, dpi=150, facecolor=BG)
print("wrote", out)
