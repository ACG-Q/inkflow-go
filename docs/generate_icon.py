"""Generate inkflow icon as ICO file."""
from PIL import Image, ImageDraw
import math

def create_icon(size=128):
    """Create inkflow icon at given size."""
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # Rounded rectangle background
    margin = int(size * 0.04)
    radius = int(size * 0.22)
    draw.rounded_rectangle(
        [margin, margin, size - margin, size - margin],
        radius=radius,
        fill=(44, 95, 45, 255)  # #2c5f2d
    )

    # Ink drop body (ellipse)
    cx, cy = size // 2, int(size * 0.44)
    rx, ry = int(size * 0.16), int(size * 0.20)
    draw.ellipse([cx - rx, cy - ry, cx + rx, cy + ry], fill=(255, 255, 255, 242))

    # Ink drop tip (brush stroke top)
    tip_top = int(size * 0.23)
    tip_bottom = int(size * 0.38)
    tip_cx = size // 2
    for y in range(tip_top, tip_bottom):
        t = (y - tip_top) / (tip_bottom - tip_top)
        w = int(size * 0.05 * (1 - t) + size * 0.01 * t)
        alpha = int(180 * (1 - t * 0.3))
        draw.line([(tip_cx - w, y), (tip_cx + w, y)], fill=(255, 255, 255, alpha))

    # Bottom ink splash
    splash_y = int(size * 0.64)
    splash_w = int(size * 0.14)
    for i in range(5):
        sx = size // 2 + int((i - 2) * size * 0.06)
        sy = splash_y + int(math.sin(i * 1.2) * size * 0.03)
        r = int(size * 0.025 * (1.2 - abs(i - 2) * 0.2))
        alpha = int(120 - abs(i - 2) * 20)
        draw.ellipse([sx - r, sy - r, sx + r, sy + r], fill=(255, 255, 255, alpha))

    # Small ink dots
    dots = [(0.37, 0.72, 0.024, 100), (0.63, 0.69, 0.016, 77)]
    for dx, dy, dr, alpha in dots:
        x, y = int(size * dx), int(size * dy)
        r = int(size * dr)
        draw.ellipse([x - r, y - r, x + r, y + r], fill=(255, 255, 255, alpha))

    return img

def main():
    img = create_icon(256)

    # Save as ICO with multiple sizes
    sizes = [(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
    imgs = [img.resize(s, Image.LANCZOS) for s in sizes]

    # Save ICO
    imgs[0].save(
        'favicon.ico',
        format='ICO',
        sizes=[(s.width, s.height) for s in imgs],
        append_images=imgs[1:]
    )
    print('favicon.ico created')

    # Also save a 256x256 PNG for go-winres
    img.save('favicon.png')
    print('favicon.png created')

if __name__ == '__main__':
    main()
