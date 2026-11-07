"""Render the neutral fork icon and brand logo with Pillow (no upstream assets).

Run: python Source/tools/render_assets.py
Outputs:
  Source/assets/RepublicModManagerRUS.ico  (16/32/48/64/128/256)
  Source/assets/brand_logo.png             (512x512, transparent)
  Source/assets/icon_rus_source.png        (1024x1024 master, PNG)
"""

from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / "assets"

BG = (18, 21, 27, 255)          # dark charcoal
RING = (70, 76, 86, 255)        # grey ring
RING_DARK = (48, 53, 61, 255)
ACCENT = (203, 32, 39, 255)     # warm red
ACCENT_DARK = (150, 22, 28, 255)
STAR = (247, 181, 14, 255)      # yellow
LETTER = (240, 240, 238, 255)


def gear_mask(size: int) -> Image.Image:
    mask = Image.new("L", (size, size), 0)
    draw = ImageDraw.Draw(mask)
    cx = cy = size / 2
    outer = int(size * 0.46)
    inner = int(size * 0.40)
    teeth = 12
    step = (2 * 3.14159265) / (2 * teeth)
    points = []
    import math
    for i in range(2 * teeth):
        angle = i * step
        radius = outer if i % 2 == 0 else inner
        for k in range(2):
            a = angle + k * (step / 2)
            points.append((cx + radius * math.cos(a), cy + radius * math.sin(a)))
    draw.polygon(points, fill=255)
    draw.ellipse((cx-inner-1, cy-inner-1, cx+inner+1, cy+inner+1), fill=0)
    return mask


def draw_icon(size: int, transparent: bool = False) -> Image.Image:
    icon = Image.new("RGBA", (size, size), (0, 0, 0, 0) if transparent else BG)
    draw = ImageDraw.Draw(icon)
    if not transparent:
        draw.rounded_rectangle((0, 0, size - 1, size - 1), radius=int(size * 0.18), fill=BG)
    ring = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    gear = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    gear.paste(Image.new("RGBA", (size, size), RING), mask=gear_mask(size))
    inner = int(size * 0.40)
    cx = cy = size / 2
    # dimple under the ring
    dimple = ImageDraw.Draw(gear)
    dimple.ellipse((cx-inner, cy-inner, cx+inner, cy+inner), fill=BG)
    dimple.ellipse((cx-inner+2, cy-inner+2, cx+inner-2, cy+inner-2), fill=RING_DARK)
    icon.alpha_composite(gear)
    # yellow star, upper-left inside the ring
    starR = int(size * 0.06)
    sx, sy = int(size * 0.30), int(size * 0.24)
    import math
    pts = []
    for i in range(8):
        radius = starR if i % 2 == 0 else starR * 0.42
        a = -math.pi/2 + i * math.pi / 4
        pts.append((sx + radius * math.cos(a), sy + radius * math.sin(a)))
    draw.polygon(pts, fill=STAR)
    # letter R
    font_size = int(size * 0.46)
    font_path = Path("C:/Windows/Fonts/arialbd.ttf")
    font = ImageFont.truetype(str(font_path), font_size)
    text = "R"
    bbox = draw.textbbox((0, 0), text, font=font)
    tw, th = bbox[2] - bbox[0], bbox[3] - bbox[1]
    # shadow
    draw.text((cx - tw/2 - bbox[0] + max(2, size*0.006), cy - th/2 - bbox[1] + max(2, size*0.012)), text, font=font, fill=ACCENT_DARK)
    draw.text((cx - tw/2 - bbox[0], cy - th/2 - bbox[1]), text, font=font, fill=ACCENT)
    return icon


def main():
    icon = draw_icon(256)
    icon_path = ASSETS / "icon_rus_source.png"
    icon.save(icon_path)

    sizes = [(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
    ico_frames = [draw_icon(px) for px, _ in sizes]
    ico_frames[0].save(ASSETS / "RepublicModManagerRUS.ico", append_images=ico_frames[1:], format="ICO", sizes=sizes)

    logo = draw_icon(512, transparent=True)
    lw = logo.width
    canvas = Image.new("RGBA", (lw, lw), (0, 0, 0, 0))
    canvas.alpha_composite(logo)
    canvas.save(ASSETS / "brand_logo.png")
    print("assets rendered")


if __name__ == "__main__":
    main()
