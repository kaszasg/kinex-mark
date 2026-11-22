"""
Graphics generator using Google Gemini.

Usage:
  set GEMINI_API_KEY=your-api-key
  python generate_graphics.py                 # everything that is still missing
  python generate_graphics.py info-tickets    # only the named images
  python generate_graphics.py --force hero    # regenerate even if it exists
  python generate_graphics.py --no-deliver    # generate + trim, skip the client copy
  python generate_graphics.py --list          # what is defined and what exists

Run it with plain `python` (not `python -I`): google-genai lives in the user site.

"""

import argparse
import json
import os
import shutil
import sys
import time
from pathlib import Path

from google import genai
from google.genai import types

BASE_DIR = Path(__file__).parent
OUTPUT_DIR = BASE_DIR
PROMPTS_FILE = BASE_DIR / "prompts.json"
CLIENT_IMAGES_DIR = BASE_DIR.parent / "web" / "static" / "images"

API_KEY_VARIABLE = "GEMINI_API_KEY"
DEFAULT_MODEL = "gemini-3.1-flash-image"

# Values the Gemini image models accept (google-genai types.ImageConfig).
ASPECT_RATIOS = ("1:1", "2:3", "3:2", "3:4", "4:3", "9:16", "16:9", "21:9")
IMAGE_SIZES = ("1K", "2K", "4K")


def parse_hex(value) -> tuple[int, int, int] | None:
    """Parses '#rrggbb' into an RGB tuple; returns None for anything else."""
    if not isinstance(value, str) or len(value) != 7 or not value.startswith("#"):
        return None
    try:
        return tuple(int(value[i:i + 2], 16) for i in (1, 3, 5))
    except ValueError:
        return None


def load_specs() -> tuple[str, dict, list[dict]]:
    """Reads prompts.json and resolves every image's style reference."""
    try:
        config = json.loads(PROMPTS_FILE.read_text(encoding="utf-8"))
    except FileNotFoundError:
        sys.exit(f"ERROR: {PROMPTS_FILE.name} is missing next to the script.")
    except json.JSONDecodeError as e:
        sys.exit(f"ERROR: {PROMPTS_FILE.name} is not valid JSON: {e}")

    model = config.get("model", DEFAULT_MODEL)
    styles = config.get("styles", {})
    specs = []

    for index, image in enumerate(config.get("images", []), 1):
        name = image.get("name")
        prompt = image.get("prompt")
        if not name or not prompt:
            sys.exit(f"ERROR: image #{index} in {PROMPTS_FILE.name} needs a name and a prompt.")

        style_name = image.get("style")
        if style_name and style_name not in styles:
            sys.exit(f"ERROR: image '{name}' references the unknown style '{style_name}'.")

        aspect_ratio = image.get("aspect_ratio")
        if aspect_ratio is not None and aspect_ratio not in ASPECT_RATIOS:
            sys.exit(f"ERROR: image '{name}' has an unsupported aspect_ratio '{aspect_ratio}' "
                     f"(allowed: {', '.join(ASPECT_RATIOS)}).")
        image_size = image.get("image_size")
        if image_size is not None and image_size not in IMAGE_SIZES:
            sys.exit(f"ERROR: image '{name}' has an unsupported image_size '{image_size}' "
                     f"(allowed: {', '.join(IMAGE_SIZES)}).")

        background = image.get("background")
        if background is not None and parse_hex(background) is None:
            sys.exit(f"ERROR: image '{name}' has an invalid background '{background}' "
                     "(expected #rrggbb).")

        size = image.get("size")
        specs.append({
            "name": name,
            "trim": bool(image.get("trim")),
            "aspect_ratio": aspect_ratio,
            "image_size": image_size,
            "background": background,
            "prompt": "".join([
                styles.get(style_name, ""),
                f"The image should be {size}. " if size else "",
                prompt,
            ]),
        })

    if not specs:
        sys.exit(f"ERROR: {PROMPTS_FILE.name} defines no images.")

    return model, config.get("delivery", {}), specs


def trim_borders(path: Path, margin: float = 0.02, tolerance: int = 6) -> None:
    """
    Crops the near-uniform border the model tends to leave around the artwork and
    puts back an even margin. Idempotent: an already trimmed image is left alone.
    """
    from PIL import Image, ImageChops

    with Image.open(path) as image:
        rgb = image.convert("RGB")
        # The corner pixel is the background by construction (flat, very light canvas).
        background = Image.new("RGB", rgb.size, rgb.getpixel((0, 0)))
        difference = ImageChops.difference(rgb, background).convert("L")
        box = difference.point(lambda value: 255 if value > tolerance else 0).getbbox()
        if box is None:
            return

        pad = round(min(rgb.size) * margin)
        left, top, right, bottom = box
        box = (max(left - pad, 0), max(top - pad, 0),
               min(right + pad, rgb.width), min(bottom + pad, rgb.height))
        if box == (0, 0, rgb.width, rgb.height):
            return

        rgb.crop(box).save(path)


def crop_to_ratio(image, aspect_ratio: str | None):
    """
    Centre-crops an image to the exact requested aspect ratio. The models return sizes
    that are only close to the nominal ratio (for example 2752 x 1536 for 16:9), so
    without this a 1600 px wide copy would not be exactly 16:9. Only the few surplus
    pixels on one axis are removed.
    """
    if not aspect_ratio:
        return image
    ratio_w, ratio_h = (int(part) for part in aspect_ratio.split(":"))
    width, height = image.size
    if width * ratio_h > height * ratio_w:
        new_width, new_height = round(height * ratio_w / ratio_h), height
    else:
        new_width, new_height = width, round(width * ratio_h / ratio_w)
    if (new_width, new_height) == (width, height):
        return image
    left = (width - new_width) // 2
    top = (height - new_height) // 2
    return image.crop((left, top, left + new_width, top + new_height))


def normalize_background(image, target_hex: str, border: float = 0.02, tolerance: int = 4):
    """
    Pulls the model's background onto the exact design token. The models paint a dark
    canvas that is close to, but not exactly, the requested colour and add a little
    noise. The measured background (median of the outer border strip) is mapped onto
    the target per channel with a monotonic piecewise-linear curve (black and white stay
    fixed, so line colours keep their contrast), then every pixel within `tolerance` of
    the measured background is set to the target exactly, which makes the canvas flat.
    """
    from PIL import Image, ImageChops

    target = parse_hex(target_hex)
    width, height = image.size
    strip = max(1, round(min(width, height) * border))
    pixels = []
    for box in ((0, 0, width, strip), (0, height - strip, width, height),
                (0, 0, strip, height), (width - strip, 0, width, height)):
        pixels.extend(image.crop(box).get_flattened_data()
                      if hasattr(image, "get_flattened_data") else image.crop(box).getdata())
    measured = tuple(sorted(pixel[channel] for pixel in pixels)[len(pixels) // 2]
                     for channel in range(3))

    def curve(source: int, goal: int) -> list[int]:
        table = []
        for value in range(256):
            if value <= source:
                mapped = value * goal / source if source else goal
            else:
                mapped = goal + (value - source) * (255 - goal) / (255 - source)
            table.append(min(255, max(0, round(mapped))))
        return table

    bands = [band.point(curve(measured[i], target[i])) for i, band in enumerate(image.split())]
    mapped = Image.merge("RGB", bands)

    difference = ImageChops.difference(image, Image.new("RGB", image.size, measured)).split()
    distance = ImageChops.lighter(ImageChops.lighter(difference[0], difference[1]), difference[2])
    mask = distance.point(lambda value: 255 if value <= tolerance else 0)
    mapped.paste(Image.new("RGB", image.size, target), mask=mask)
    return mapped, measured


def deliver(source: Path, target_dir: Path, delivery: dict, aspect_ratio: str | None = None,
            background: str | None = None) -> Path:
    """
    Writes the web copy of an image. The PNG in graphics/ stays the master; the client
    gets the delivery format, which for photographic-looking illustrations is an order of
    magnitude smaller (~3.4 MB of infographics become ~290 KB).

    With an image `background` the copy's canvas is normalised onto that colour first.
    With delivery.max_width the copy is cropped to the image's exact aspect ratio and
    scaled down proportionally (Lanczos) to that width. A smaller image is not enlarged.
    The master PNG is never modified here.
    """
    fmt = delivery.get("format", "png").lower()
    max_width = delivery.get("max_width")
    if fmt == "png" and not max_width and not background:
        shutil.copy2(source, target_dir / source.name)
        return target_dir / source.name

    from PIL import Image

    options = {}
    if fmt in ("webp", "jpeg", "jpg"):
        options["quality"] = delivery.get("quality", 88)
    if fmt == "webp":
        options["method"] = 6  # slowest, smallest — this runs once per image, offline

    target = target_dir / f"{source.stem}.{fmt}"
    with Image.open(source) as image:
        copy = image.convert("RGB")
        if background:
            copy, measured = normalize_background(copy, background)
            print(f"  {target.name:<22} background #{bytes(measured).hex()} -> {background}")
        if max_width:
            copy = crop_to_ratio(copy, aspect_ratio)
            if copy.width > max_width:
                height = round(copy.height * max_width / copy.width)
                copy = copy.resize((max_width, height), Image.LANCZOS)
        copy.save(target, "JPEG" if fmt == "jpg" else fmt.upper(), **options)
    return target


def generate_image(client, model: str, spec: dict, force: bool) -> bool:
    """Generates a single image and saves it. Returns True when the file exists."""
    name = spec["name"]
    output_path = OUTPUT_DIR / f"{name}.png"

    if output_path.exists() and not force:
        print(f"  [SKIP] {name}.png already exists")
        return True

    image_options = {key: spec[key] for key in ("aspect_ratio", "image_size") if spec.get(key)}
    config = types.GenerateContentConfig(
        response_modalities=["IMAGE"],
        image_config=types.ImageConfig(**image_options) if image_options else None,
    )

    print(f"  [GEN]  {name}.png ...", end=" ", flush=True)
    try:
        response = client.models.generate_content(model=model, contents=[spec["prompt"]],
                                                  config=config)

        for part in response.parts or []:
            if part.inline_data is not None:
                part.as_image().save(str(output_path))
                version = getattr(response, "model_version", None) or model
                print(f"OK (model version: {version})")
                return True

        print("WARN: no image in response")
        for part in response.parts or []:
            if part.text:
                print(f"         Model said: {part.text[:200]}")

    except Exception as e:
        print(f"ERROR: {e}")

    return output_path.exists()


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[1])
    parser.add_argument("names", nargs="*", help="only these images (default: all of them)")
    parser.add_argument("--force", action="store_true", help="regenerate existing images too")
    parser.add_argument("--list", action="store_true", help="list the defined images and exit")
    parser.add_argument("--no-deliver", action="store_true",
                        help="skip copying the web versions into the client project")
    args = parser.parse_args()

    model, delivery, specs = load_specs()

    if args.names:
        known = {spec["name"] for spec in specs}
        unknown = [name for name in args.names if name not in known]
        if unknown:
            sys.exit(f"ERROR: unknown image name(s): {', '.join(unknown)}")
        specs = [spec for spec in specs if spec["name"] in args.names]

    if args.list:
        print(f"Model: {model}")
        for spec in specs:
            state = "present" if (OUTPUT_DIR / f"{spec['name']}.png").exists() else "missing"
            shape = " ".join(spec.get(key) or "-" for key in ("aspect_ratio", "image_size"))
            print(f"  {spec['name']:<16} {state:<8} {shape}")
        return

    api_key = os.environ.get(API_KEY_VARIABLE, "")
    if not api_key:
        sys.exit(f"ERROR: Set the {API_KEY_VARIABLE} environment variable first.")

    client = genai.Client(api_key=api_key)

    print(f"Generating {len(specs)} image(s) with {model}...")
    print(f"Output directory: {OUTPUT_DIR}\n")

    success = 0
    for i, spec in enumerate(specs, 1):
        print(f"[{i}/{len(specs)}]", end=" ")
        if generate_image(client, model, spec, args.force):
            success += 1
        if i < len(specs):
            time.sleep(2)

    # Normalize the framing, then publish the images next to the client app. Both steps
    # run for existing files too, so neither is tied to a (paid) regeneration.
    for spec in specs:
        source = OUTPUT_DIR / f"{spec['name']}.png"
        if source.exists() and spec["trim"]:
            trim_borders(source)

    print(f"\nDone! {success}/{len(specs)} image(s) available.")

    if args.no_deliver:
        print("Delivery skipped (--no-deliver).")
    elif not CLIENT_IMAGES_DIR.parent.is_dir():
        # web/static missing means the SvelteKit project does not exist (yet): do not create it.
        print(f"Delivery skipped: {CLIENT_IMAGES_DIR.parent} does not exist.")
        print("Hint: create the web/static folder first (SvelteKit project), then re-run the script "
              "(existing images are not regenerated, only trimmed and delivered).")
    else:
        from PIL import Image

        CLIENT_IMAGES_DIR.mkdir(exist_ok=True)
        for spec in specs:
            source = OUTPUT_DIR / f"{spec['name']}.png"
            if source.exists():
                target = deliver(source, CLIENT_IMAGES_DIR, delivery, spec.get("aspect_ratio"),
                                 spec.get("background"))
                with Image.open(target) as image:
                    print(f"  {target.name:<22} {image.width} x {image.height} px, "
                          f"{target.stat().st_size / 1024:.0f} KB")
        print(f"Copied to: {CLIENT_IMAGES_DIR}")

    if success < len(specs):
        print("\nTip: re-run the script to retry the failures (existing images are skipped).")
        sys.exit(1)


if __name__ == "__main__":
    main()
