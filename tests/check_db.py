import sqlite3

conn = sqlite3.connect("C:/Users/LiuJi/Desktop/test4/inkflow-go/data/inkflow.db")
c = conn.cursor()
c.execute("SELECT name FROM sqlite_master WHERE type='table'")
tables = [r[0] for r in c.fetchall()]
print("Tables:", tables)

if "template_controls" in tables:
    c.execute("PRAGMA table_info(template_controls)")
    print("Columns:", [r[1] for r in c.fetchall()])
    c.execute("SELECT COUNT(*) FROM template_controls")
    print("Rows:", c.fetchone()[0])
else:
    print("template_controls table NOT FOUND")
    # Check what the actual table name is
    for t in tables:
        if "control" in t.lower():
            print(f"  Found: {t}")
            c.execute(f"PRAGMA table_info({t})")
            print(f"  Columns: {[r[1] for r in c.fetchall()]}")
