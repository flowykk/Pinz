"""Generate small synthetic datasets for demos.

This script prefers Parquet output (pyarrow/fastparquet) but will fall back
to CSV + image files when Parquet support is unavailable in the environment.

Outputs placed in the same folder as this script (ML/data/).
"""
from pathlib import Path
import base64
import csv
from io import BytesIO
from PIL import Image

try:
    import pandas as pd
    HAS_PANDAS = True
except Exception:
    HAS_PANDAS = False

ROOT = Path(__file__).resolve().parent
DATA_DIR = ROOT

texts = [
    "This is a harmless travel post about the beach.",
    "Ski trip in the mountains, amazing views!",
    "Buy now! Visit spammy-site.com",
    "This contains a badword in the content",
    "Lovely city trip with food and museums",
]
labels = [0, 0, 1, 1, 0]  # 0=clean, 1=violating

def write_parquet_text(df, path):
    try:
        df.to_parquet(path)
        print(f'Wrote {path} ({len(df)} rows)')
        return True
    except Exception as e:
        print('Parquet write failed:', e)
        return False

def write_parquet_images(df, path):
    try:
        df.to_parquet(path)
        print(f'Wrote {path} ({len(df)} rows)')
        return True
    except Exception as e:
        print('Parquet write failed:', e)
        return False

# --- Text dataset ---
text_rows = [{'text': t, 'label': l} for t, l in zip(texts, labels)]

if HAS_PANDAS:
    import pandas as pd
    text_df = pd.DataFrame(text_rows)
    text_path = DATA_DIR / 'text_dataset.parquet'
    if not write_parquet_text(text_df, text_path):
        # fallback to CSV
        csv_path = DATA_DIR / 'text_dataset.csv'
        text_df.to_csv(csv_path, index=False)
        print(f'Wrote {csv_path} ({len(text_df)} rows)')
else:
    # write CSV fallback without pandas
    csv_path = DATA_DIR / 'text_dataset.csv'
    with open(csv_path, 'w', newline='', encoding='utf-8') as f:
        writer = csv.DictWriter(f, fieldnames=['text', 'label'])
        writer.writeheader()
        for r in text_rows:
            writer.writerow(r)
    print(f'Wrote {csv_path} ({len(text_rows)} rows)')

# --- Image dataset ---
colors = [(255,0,0), (0,255,0), (0,0,255), (255,255,0), (0,255,255)]
rows = []
img_folder = DATA_DIR / 'images'
img_folder.mkdir(exist_ok=True)
for i, col in enumerate(colors):
    img = Image.new('RGB', (64 + i*10, 64 + i*5), color=col)
    buf = BytesIO()
    img.save(buf, format='PNG')
    b = buf.getvalue()
    label = 0 if i % 2 == 0 else 1
    filename = f'image_{i}.png'
    (img_folder / filename).write_bytes(b)
    rows.append({'image_path': str(img_folder / filename), 'label': label, 'width': img.width, 'height': img.height})

if HAS_PANDAS:
    import pandas as pd
    img_df = pd.DataFrame(rows)
    img_path = DATA_DIR / 'images_dataset.parquet'
    if not write_parquet_images(img_df, img_path):
        csv_path = DATA_DIR / 'images_dataset.csv'
        img_df.to_csv(csv_path, index=False)
        print(f'Wrote {csv_path} ({len(img_df)} rows)')
else:
    csv_path = DATA_DIR / 'images_dataset.csv'
    with open(csv_path, 'w', newline='', encoding='utf-8') as f:
        writer = csv.DictWriter(f, fieldnames=['image_path', 'label', 'width', 'height'])
        writer.writeheader()
        for r in rows:
            writer.writerow(r)
    print(f'Wrote {csv_path} ({len(rows)} rows)')

print('Synthetic datasets generated.')
