# Character sprite rules — reasoning and corrections

Read this when a decision in SKILL.md feels underdetermined, when the user
challenges a choice, or when you are reviewing existing pixel art and need to
explain *why* something is wrong.

Evidence grades used below:
- **[source]** confirmed in a primary tutorial (Derek Yu, Slynyrd original text, Lospec)
- **[reasoned]** agreed through adversarial review, no primary source checked
- **[uncalibrated]** a defensible starting number that has not been validated against a corpus
- **[myth]** widely repeated and wrong; listed so you can recognize and correct it

## Contents

1. Workflow order and where it breaks down
2. Resolution and pixel density
3. Lines and limbs
4. Shading and the three defects
5. Color: counts, ramps, hue direction, color space
6. Outlines
7. Anti-aliasing
8. Dithering
9. Animation
10. Myths to correct on sight

---

## 1. Workflow order and where it breaks down

Silhouette → flats → shading → detail. **[source]**

The reason detail-last matters: detail attached to a broken pose is wasted work,
and worse, it hides the broken pose from you. An hour on a face will make you
defend the body it is attached to.

Where it breaks: at 8x8 and 16x16 the stages collapse, because a single pixel
change is simultaneously a silhouette edit, a shading edit and a detail edit.
Small-sprite practice is iterative sculpting — place, evaluate, move — and often
starts from a color cluster or a landmark (the head, the eyes) rather than a
monochrome silhouette. **[reasoned]** Treat the staged order as binding at 32px+
and as a loose guide below that.

The squint test — step back, and a few large light/dark clusters should still
read — is the check that the form survived the detail. **[source]** Its
programmatic equivalent is: downscale, quantize to 2–3 lightness bands, and
confirm large contiguous clusters remain. You can approximate it by exporting a
PNG and viewing it small.

## 2. Resolution and pixel density

16x16 is a good default for speed and readability, not a rule. The real drivers
are tile grid, how much screen the sprite must occupy at the target base
resolution, and how much the sprite has to communicate.

**Outline cost rises monotonically as sprites shrink.** As a fraction of opaque
pixels a 1px outline is roughly 40% at 16x16, 20% at 32x32, negligible at 64+.
**[reasoned]** So small sprites are where you drop outlines in favor of color
boundaries, and large sprites are where a full outline is affordable. Derek Yu's
tutorial makes this point about 32x32 because in his frame — his own art lives
at 64–80px — 32x32 *is* the small end. **[source]** State the gradient rather
than attaching the warning to one size.

**Mixels are the actual density failure.** Mixing asset *dimensions* is normal —
a 48px door among 16px tiles is fine. Mixing texel *density* is not: one asset
drawn 1:1 beside another drawn at 2x and downscaled, or hi-res UI over low-res
sprites. **[reasoned]** Every asset in a set shares one pixel size on screen.

## 3. Lines and limbs

Minimize jaggies — single pixels or short segments that break a line's flow.
Keep outlines 1px. On curves, segment lengths should grow or shrink
monotonically (4-3-2-1); uneven progression is what reads as a jaggy. Avoid
doubles and triples (repeated equal-length segments) in a curve, which flatten
its curvature. **[source]**

These are 32px+ rules. At 16x16 a line is 2–4 pixels total, there is no
"progression" to speak of, and doubles are unavoidable — applying the rule there
rejects correct sprites. **[reasoned]**

**Limb thickness, corrected.** The tutorial rule "never make limbs 1px" is
**[myth]** at small sizes: Mega Man's legs, Pokémon overworld arms, Stardew's
farmer and Celeste's Madeline all use 1px limbs. The rule that actually explains
those sprites: **[reasoned]**

| outline policy | minimum limb width | why |
|---|---|---|
| full outline | `2 * outline + 1` (3px at 1px outline) | outline occupies both sides of the cross-section; 1px is left for fill |
| no-fill limb | 1px | the outline color *is* the limb; legal for short limbs (~4px) or when the silhouette carries the read |
| selective outline | 2px | outlined on one side only |

Pick a branch per limb, deliberately. The failure to avoid is a limb that was
supposed to hold a fill color and does not.

Related **[source]** warning: "cardboard" designs, where the grid pulls
everything onto straight lines and the character goes stiff. The counter is to
think in 3D forms and exaggerate the characteristic features. No mechanical
check exists for this one.

## 4. Shading and the three defects

One light source, positioned above and slightly in front, is the safe default.
**[source]** Polished modern pixel art routinely adds a rim or bounce light
(Blasphemous, Eastward, Owlboy) — "one light" is a guardrail, not a ceiling.
**[reasoned]**

**Pillow shading** — shading inward from the outline in concentric even rings.
Almost never occurs physically; makes the form blurry and directionless.
**[source]** Recognizable by shadows that follow the silhouette at uniform
distance.

**Cluster banding** — two color regions whose boundary runs parallel and
adjacent along its length, reading as blur instead of as an edge. Fix by
reshaping the clusters so the boundaries diverge. **[source]**

One important exception: a 1px darker line along the inside bottom edge of a
sprite — ground-contact shading — is technically cluster banding and is standard
practice. The defect is the unintentional parallel hug, not every parallel edge.
**[reasoned]**

**Noise** — scattered isolated pixels that belong to no cluster. Group them.
Exception: deliberate texture on dirt, grass, rough stone. **[source]**

A separate defect shares the name "banding": **gradient banding**, visible
lightness steps across a large flat area, fixed with dithering or an extra ramp
step. These are different problems with different fixes; keeping the names apart
matters if you are ever critiquing someone's work. **[reasoned]**

## 5. Color

### Counts

The frequently quoted "4–8 colors per sprite" matches no actual console.
**[myth as hardware]** For reference, real constraints were: NES 3 colors +
transparent per sprite tile (which is why Mega Man's face is a separate overlaid
sprite), SNES 15 + transparent per sprite palette, GBA 16×16-color or one
256-color OBJ palette, Genesis 15×4. **[reasoned — not re-checked against
hardware manuals]**

Likewise "16–32 colors per game" comes from palette culture — DawnBringer's
DB16/DB32, PICO-8's fixed 16, Endesga 32 — not from shipping requirements.
Celeste and Owlboy use hundreds of colors; Downwell uses about three.
**[reasoned]**

So treat counts as a style dial with a defensible default rather than a law:

- master palette ~32: about 7 material ramps (skin, foliage, stone/earth, wood,
  metal, sky/water, accent) × 4 steps, plus a shared near-black outline family
  and 1–2 shared lights. Below ~32 a palette starts forcing material-identity
  errors (wood and skin sharing a ramp); much above ~48 and consistency decays
  because mistakes have room to hide. **[uncalibrated]**
- per sprite: 16x16 ≤ 6, 32x32 ≤ 10, 64x64 ≤ 16, including the outline.
  The basis is cluster economics — a color that cannot own a cluster of ~4px does
  not read, and a 16x16 sprite has only ~140–190 opaque pixels. **[uncalibrated]**

### Ramps

Build ramps, don't freehand-pick. A ramp holding hue constant and varying only
brightness — a "straight ramp" — drifts toward gray and reads muddy. Varying hue
along with lightness and saturation is what reads as form. **[source]**

**Direction is set by the light, not by a rule.** Highlights shift toward the key
light's hue; shadows shift toward the ambient or fill hue. **[reasoned]** In
daylight — warm sun, blue sky ambient — that gives warm highlights and cool
shadows, which is the version tutorials state as universal. It inverts under
moonlight, underwater, a torch behind the subject, neon, or magic light: cool
highlights, warm shadows. In fantasy games that inverted case is most scenes,
so the tutorial version stated as a law is **[myth]**.

Magnitude is also a choice. Aggressive hue shifting is itself a recognizable
tutorial-culture look; classic Capcom and Konami sprites shift far less.
**[reasoned]**

### Color space

**Step lightness in OKLCH or CIELAB, not HSB.** HSB's Brightness is not
perceptual lightness, so uniform B steps give visibly uneven ramps — yellow at
B=100 is far brighter than blue at B=100. Every ramp, hue shift and contrast
judgment inherits this error if you reason in HSB. **[reasoned]** This is the
single highest-leverage correction in this document.

### On Slynyrd's palette numbers

Slynyrd's Pixelblog 1 does literally state 8 ramps × 9 swatches = 128 colors,
+20° hue per swatch, 45° between ramps, brightness rising left to right,
saturation peaking mid-ramp. **[source]** Do not execute those numbers:

- 9 swatches at +20° sweeps 160° of hue within one ramp — a ramp starting navy
  ends yellow-green. That is his signature landscape look, not a norm.
- It is HSB-based, so it inherits the perceptual problem above.
- 45°-spaced anchors guarantee ramps land on arbitrary hues rather than the hues
  a character needs (skin, metal, cloth).
- It is a master palette for full-scene landscape mockups, not a sprite palette.
- The source itself hedges: the hue is "somewhat arbitrary", it "comes down to
  trusting the eyeballs", and he suggests starting with a small ramp instead.

Cite it as an example of ramp construction. Don't cite the numbers as a recipe.

### Other color rules **[source]**

- Too many near-identical colors makes pixels blend and disappear. Cut colors, raise contrast.
- Naive coloring — pure green leaves, pure gray rock — ignores reflected and
  ambient light. Mix environment color in.
- Sharing an end color between ramps ties a palette together.
- A careful palette swap can make assets from different sources look like one artist.

## 6. Outlines

**Default to a full closed dark outline for game sprites.** The reason is
situational, not aesthetic: a game sprite must read against arbitrary, animated
backgrounds, which is why Nuclear Throne, Rivals of Aether, Shovel Knight and
classic Capcom/SNK all use dark outlines. **[reasoned]**

Concrete spec: 1px, closed, colored as the sprite's dominant hue carried into
near-black — OKLCH L roughly 0.12–0.22, chroma roughly 0.03–0.06, hue borrowed
from the sprite and optionally nudged toward the scene's ambient. Pure `#000000`
is the fallback: never wrong for readability, only for cohesion.
**[uncalibrated]**

Contrast target: ΔL of at least about 0.30 in OKLab between the outline and the
median lightness of the background it sits on. **[uncalibrated]**

The trick for unknown backgrounds is to constrain the *other* side: if
background and tile output is kept to mid lightness (roughly L 0.35–0.75) and
the lightness extremes are reserved for gameplay elements, a near-black outline
clears that contrast target against every legal background by construction. This
is also why shipped games get away with dark outlines — their background artists
were obeying a value-band budget implicitly. **[reasoned]**

**Selective outlining** — replacing the outline with lighter color where light
hits, dropping it entirely where the sprite meets negative space, using material
shadow colors for interior partitions — is a real technique **[source]** and a
real style. It was also a 2000s pixelation-forum fad that the same community
later half-disowned when it was overused. **[reasoned]** Use it when the user
asks or when you control the background.

Interior partitions should use each material's shadow step, never the outline
color, so the outline retains one unambiguous role. **[source]**

Consistency is checkable and matters: a sprite outlined on half its perimeter
reads as a decision nobody made.

## 7. Anti-aliasing

Interior AA works: place an intermediate color at corners where segments meet,
scale the AA run to the segment length, favor angles near 45°, and choose AA
colors close to the two colors they sit between. **[source]**

**Do not AA the outer silhouette.** The reason usually given — "it will look
dirty on an unknown background" — is a fixed-palette, no-alpha argument that no
longer holds; a modern GPU blends the edge arithmetically against the real
background, producing a correct local intermediate. **[reasoned]** The reasons
that do hold:

1. **Pipeline.** Palette-locked output, palette-swap shaders, index-based
   effects and 1-bit-alpha pipelines all break on semi-transparent pixels. An
   indexed Aseprite workflow is exactly this case.
2. **Style consistency.** At 4x–6x display scale the sprite gets one ring of soft
   pixels around an otherwise hard interior, which reads as fringe, not smoothing.
3. **Motion.** The blend recomputes every frame against a scrolling background,
   so the silhouette shimmers while the interior stays locked.

Do not justify it with "there is no CRT to blur it" — that argument belongs to
dithering (which depends on optical mixing) and does not transfer to alpha
(which is arithmetic). Getting the rationale right matters when you explain a
correction to someone. **[reasoned]**

Related export hygiene: if anything downstream filters the texture, the RGB of
fully transparent pixels bleeds in as a dark fringe. Bleeding edge color into
transparent texels — defringe — is the standard fix. **[reasoned]**

## 8. Dithering

Dithering is a two-color noise pattern that simulates an intermediate color,
useful on large flat areas and rough texture, and it prevents gradient banding.
**[source]**

The governing variable is palette budget and display, not texture. Dithering
worked because CRT and composite video physically blurred the checkerboard into
a blend — Genesis waterfalls were designed for exactly that. On a modern LCD at
4x–6x integer scale each dither pixel is a crisp block, so the pattern reads as
texture, never as a blended color. **[reasoned]**

Practical consequence: if you have a spare palette slot, add a ramp step instead
of dithering. Reach for dithering when you specifically want retro texture or a
stylistic signature.

## 9. Animation

Timing dominates frame count: a 4-frame walk with well-chosen durations beats an
8-frame walk with uniform ones. Use variable durations — slow anticipation, fast
action, slow recovery. Layer in anticipation, follow-through, overlapping action
and secondary motion once the base cycle reads. **[source]**

Shipped frame-count reference points, useful as budgets: SMB1 walk/run 3 frames,
NES Mega Man run 3–4, Shovel Knight run 6, modern indie walk typically 6–8,
Symphony of the Night's Alucard walk 16 (cited as the lavish ceiling). Attacks
are typically 3–6 frames total: 1–2 anticipation, one contact frame that is
often the smear, 2 recovery. **[reasoned — not verified against sprite rips]**

At low resolution 1–3px of movement is enough, and a 1px squash before a jump
plus a 1px stretch at the apex removes robotic stiffness. At 4x–6x display scale
1px is 4–6 screen pixels; held two frames it clearly reads. **[reasoned]**

Smear frames sell speed with a single distorted pose that does not need to look
correct frozen. The classic Castlevania whip is often cited as three animation
cells with the middle one acting as the smear — treat that specific claim as
unverified; it traces to a low-quality secondary source. **[uncalibrated]**

Worth knowing even though this skill bakes frames: Celeste implements squash and
stretch as *runtime* non-uniform sprite scaling on a ~16px character, applied
inside its 320x180 render target before the whole target is upscaled — so the
sprite is re-rasterized onto the world grid and every pixel reaching the screen
is still a uniform block. **[reasoned — source not read directly]** If a user
asks why their engine can squash sprites without breaking the grid, that is the
mechanism. Bake frames instead when the art must survive static inspection.

## 10. Myths to correct on sight

| Claim you will hear | Reality |
|---|---|
| "Never use 1px limbs" | Standard at 16x16 in shipped games; the real constraint is `2 * outline + 1` if the limb must hold fill |
| "Shadows always cool, highlights always warm" | Daylight only. Follows key light and ambient hue; inverts at night, underwater, torchlit |
| "4–8 colors per sprite is the hardware limit" | Matches no console. NES was 3 + transparent per tile; SNES 15 + transparent |
| "16–32 colors per game" | Palette-community convention (DB32, PICO-8), not a shipping requirement |
| "Export animation as GIF" | GIF is a preview format — 256 colors, 1-bit alpha, no engine imports it as an animation asset. Ship a spritesheet with metadata |
| "Always scale by integers" | True of the *final* blit. Effects and camera motion happen inside the low-res render target; sharp-bilinear absorbs a non-integer remainder |
| "Always turn mipmaps off" | True for grid-aligned 2D at fixed scale. Pixel art that rotates, zooms, or renders at distance shimmers badly without mips |
| "Anti-aliased edges look dirty on unknown backgrounds" | Modern alpha blending composites correctly. The real objections are palette pipelines, fringe at chunky scale, and motion shimmer |
| Slynyrd's 128-color recipe as a method | His personal landscape palette; the source itself calls the hue "somewhat arbitrary" |
