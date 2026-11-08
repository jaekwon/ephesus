#!/usr/bin/env python3
"""rev12.py [FROM_YEAR TO_YEAR]: when does the sky look like Revelation 12:1?

"A woman clothed with the sun, with the moon under her feet, and on her head a crown of twelve stars." The 2017
reading: the Sun in Virgo (clothing her), the Moon at her feet, Jupiter inside her for about nine months (the child),
and Mercury, Venus and Mars in Leo above her head, which with Leo's nine stars make twelve. This script scores every
day on which the Sun is in Virgo (IAU boundaries) between FROM_YEAR and TO_YEAR for those four conditions, prints the
days that meet all of them, and for each says how tight the fit is: how far the Moon sits below Virgo's feet, how many
days Jupiter had been in Virgo, and how close together the three planets stand in Leo.

Positions come from astropy's built-in ephemeris (ERFA plan94 for the planets, moon98 for the Moon): arcminute-level,
good enough to say what constellation a body is in. Geocentric, which is what a watcher sees.

Needs astropy.
"""
import sys
import warnings

import numpy as np
from astropy.coordinates import SkyCoord, get_body, get_constellation, solar_system_ephemeris
from astropy.time import Time

y0 = int(sys.argv[1]) if len(sys.argv) > 1 else 1
y1 = int(sys.argv[2]) if len(sys.argv) > 2 else 2100
solar_system_ephemeris.set("builtin")
warnings.filterwarnings("ignore")

# Virgo's feet in the classical figure: Mu Virginis (Rijl al Awwa, "the foot of the barking dog") and Iota Virginis
# (Syrma, "the train of her dress"), with Lambda Virginis, Khambalia. J2000, degrees.
FEET = SkyCoord(ra=[220.765, 214.004, 214.778], dec=[-5.658, -6.001, -13.371], unit="deg", frame="icrs")
SPICA = SkyCoord(ra=201.298, dec=-11.161, unit="deg", frame="icrs")


def const(c):
    return get_constellation(c, short_name=True)


def body(name, t):
    return get_body(name, t)


def sun_in_virgo_days(year):
    """The days of `year` on which the Sun is in Virgo (about 16 September to 30 October)."""
    t = Time(f"{year:04d}-09-01T12:00:00", scale="utc") + np.arange(0, 75) * 1.0
    s = body("sun", t)
    return t[const(s) == "Vir"]


def jupiter_days_in_virgo_before(t):
    """How many days Jupiter had been in Virgo without a break before t (0 if it is not there)."""
    if const(body("jupiter", t)) != "Vir":
        return 0
    n = 0
    step = 5
    while n < 600:
        if const(body("jupiter", t - (n + step) * 1.0)) != "Vir":
            break
        n += step
    return n


def crown(t):
    """Mercury, Venus and Mars: which constellation each is in, and their spread in degrees."""
    ps = [body(p, t) for p in ("mercury", "venus", "mars")]
    cs = [const(p) for p in ps]
    sep = max(ps[i].separation(ps[j]).deg for i in range(3) for j in range(i + 1, 3))
    return cs, sep


hits = []
for year in range(y0, y1 + 1):
    days = sun_in_virgo_days(year)
    if len(days) == 0:
        continue
    # Jupiter first: it is in Virgo about one year in twelve
    if const(body("jupiter", days[len(days) // 2])) != "Vir":
        continue
    # Mars next: in Leo about two months in two years
    mars = const(body("mars", days))
    for t, in_leo in zip(days, mars == "Leo"):
        if not in_leo:
            continue
        cs, sep = crown(t)
        if cs != ["Leo", "Leo", "Leo"]:
            continue
        moon = body("moon", t)
        feet = min(moon.separation(f).deg for f in FEET)
        if feet > 15:  # the Moon moves 13 degrees a day; within 15 of the feet is "at her feet" (in Virgo or just below)
            continue
        below = moon.dec.deg < min(FEET.dec.deg) + 2  # at or below the feet, not up at her waist
        jd = jupiter_days_in_virgo_before(t)
        hits.append((t.utc.iso[:10], feet, below, jd, sep, const(moon)))

print(f"Sun in Virgo, Moon within 15 degrees of Virgo's feet, Jupiter in Virgo, Mercury, Venus and Mars all in Leo: {len(hits)} days, {y0}-{y1}")
print("date         moon-to-feet(deg)  moon at/below feet  moon in  jupiter days in Virgo before  crown spread(deg)")
for d, feet, below, jd, sep, mc in hits:
    print(f"{d}   {feet:6.1f}             {'yes' if below else 'no '}            {mc:3s}        {jd:4d}                        {sep:5.1f}")
