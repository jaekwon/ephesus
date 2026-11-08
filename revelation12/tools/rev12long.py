#!/usr/bin/env python3
"""rev12long.py EPH_DIR [Y0 Y1] [--grade]: how unique is the sky of 23 September 2017?

Scans every day from Y0 to Y1 (default the whole span of JPL's long ephemeris DE441, 13200 BC to AD 17191) for the
Revelation 12 configuration and grades each hit against 2017:

  Sun in Virgo (IAU boundaries) and Jupiter in Virgo;
  Mercury, Venus and Mars all in Leo (with Leo's nine named stars, a crown of twelve);
  the Moon within 15 degrees of Virgo's feet (Mu, Iota, Lambda Virginis) and no higher than her ankles.

For each hit it reports how many days Jupiter had been in Virgo without a break (the "child", about nine months
in 2017), how far apart the three planets stood (a crown needs distinct points: 2017's closest pair was 6.7
degrees, its widest 14.0), where the Moon was, and whether the crown sits in the Lion's figure (IAU Leo reaches
far south of it). A hit is "2017-grade" when Jupiter had been in Virgo at least 270 days, the closest two planets
were at least 5 degrees apart, the widest pair at most 20, the lowest of the three no lower than +4 degrees of
declination, and the Moon within 13 degrees of the feet.

Positions: JPL DE441 through skyfield (de441_part-1.bsp before 1969-06-28, part-2 after), geocentric astrometric
ICRS, so the constellation test (astropy, boundaries fixed on the sky) holds at any epoch. Dates are printed in
Terrestrial Time on the proleptic Gregorian calendar; thousands of years out, the civil date of a configuration is
uncertain by days (Delta T), but the configuration itself is not. Stars' proper motion over ten thousand years
moves the figure by about a degree; ignored.

Needs skyfield, jplephem, astropy, numpy.
"""
import sys
import time as clock

import numpy as np
from astropy.coordinates import SkyCoord, get_constellation
from skyfield.api import load

args = [a for a in sys.argv[1:] if not a.startswith("--")]
eph_dir = args[0]
Y0 = int(args[1]) if len(args) > 1 else -13000
Y1 = int(args[2]) if len(args) > 2 else 17000

ts = load.timescale()
SPLIT = 2440400.5  # DE441 part 1 ends and part 2 begins on 1969-06-28
parts = {}
def eph_for(jd):
    key = 1 if jd < SPLIT else 2
    if key not in parts:
        parts[key] = load(f"{eph_dir}/de441_part-{key}.bsp")
    return parts[key]

FEET = SkyCoord(ra=[220.765, 214.004, 214.778], dec=[-5.658, -6.001, -13.371], unit="deg")
FEET_TOP = -3.5  # declination of the ankles, roughly

def radec(eph, body, t):
    ra, dec, _ = eph["earth"].at(t).observe(eph[body]).radec()
    return SkyCoord(ra=ra.hours * 15, dec=dec.degrees, unit="deg")

def const(c):
    return np.asarray(get_constellation(c, short_name=True))

def jupiter_days_before(eph, jd):
    n = 0
    while n < 700:
        t = ts.tt_jd(jd - (n + 10))
        if const(radec(eph, "jupiter barycenter", t))[()] != "Vir":
            break
        n += 10
    return n

hits = []
t_start = clock.time()
CHUNK = 120000  # days per pass, about 330 years
jd0 = ts.tt(Y0, 1, 1).tt
jd1 = ts.tt(Y1, 12, 31).tt
jd = jd0
while jd < jd1:
    end = min(jd + CHUNK, jd1)
    if jd < SPLIT < end:
        end = SPLIT  # a pass never straddles the two files
    days = np.arange(jd, end, 1.0)
    eph = eph_for(days[0])
    t = ts.tt_jd(days)
    sun = radec(eph, "sun", t)
    m = const(sun) == "Vir"
    if m.any():
        d1 = days[m]
        jup = radec(eph, "jupiter barycenter", ts.tt_jd(d1))
        m2 = const(jup) == "Vir"
        if m2.any():
            d2 = d1[m2]
            mars = radec(eph, "mars barycenter", ts.tt_jd(d2))
            m3 = const(mars) == "Leo"
            if m3.any():
                d3 = d2[m3]
                t3 = ts.tt_jd(d3)
                merc, ven = radec(eph, "mercury", t3), radec(eph, "venus", t3)
                m4 = (const(merc) == "Leo") & (const(ven) == "Leo")
                if m4.any():
                    d4 = d3[m4]
                    t4 = ts.tt_jd(d4)
                    moon = radec(eph, "moon", t4)
                    feet = np.min([moon.separation(f).deg for f in FEET], axis=0)
                    m5 = (feet <= 15) & (moon.dec.deg <= FEET_TOP)
                    for i in np.nonzero(m5)[0]:
                        tj = ts.tt_jd(d4[i])
                        ps = [radec(eph, b, tj) for b in ("mercury", "venus", "mars barycenter")]
                        seps = [ps[a].separation(ps[b]).deg for a in range(3) for b in range(a + 1, 3)]
                        cal = tj.tt_calendar()
                        hits.append({"date": f"{int(cal[0]):+06d}-{int(cal[1]):02d}-{int(cal[2]):02d}", "jd": float(d4[i]),
                                     "moon_feet": float(feet[i]), "moon_dec": float(moon.dec.deg[i]),
                                     "min_sep": min(seps), "max_sep": max(seps), "low_dec": min(float(p.dec.deg) for p in ps),
                                     "jup_days": jupiter_days_before(eph, d4[i])})
    jd = end
    if jd == SPLIT:
        jd = SPLIT + 1.0  # skip 1969-06-28 itself: light-time correction would reach before part 2 begins
    done = (jd - jd0) / (jd1 - jd0)
    print(f"\r{100*done:5.1f}%  {clock.time()-t_start:6.0f}s  hits so far: {len(hits)}", end="", file=sys.stderr, flush=True)
print(file=sys.stderr)

def grade(h):
    # IAU Leo reaches south to -6 degrees, well below the Lion's figure; a crown sits in the figure, so the lowest of
    # the three planets must be at +4 degrees or higher (2017's lowest, Mercury, was at +6.4)
    return h["jup_days"] >= 270 and h["min_sep"] >= 5 and h["max_sep"] <= 20 and h["moon_feet"] <= 13 and h["low_dec"] >= 4

print(f"Sun and Jupiter in Virgo, Mercury, Venus and Mars in Leo, Moon at Virgo's feet: {len(hits)} days in {Y0}..{Y1}")
print("date (TT)     moon-feet  moon dec  planets min/max sep  lowest planet dec  jupiter days in Virgo  2017-grade")
for h in hits:
    print(f"{h['date']}   {h['moon_feet']:5.1f}     {h['moon_dec']:6.1f}     {h['min_sep']:4.1f} / {h['max_sep']:4.1f}        {h['low_dec']:6.1f}           {h['jup_days']:4d}                 {'YES' if grade(h) else 'no'}")
good = [h for h in hits if grade(h)]
events = []
for h in good:  # consecutive days are one event
    if events and h["jd"] - events[-1][-1]["jd"] <= 3:
        events[-1].append(h)
    else:
        events.append([h])
print(f"\n2017-grade events: {len(events)} in {Y1 - Y0} years: " + ", ".join(e[0]["date"][:6] for e in events))
