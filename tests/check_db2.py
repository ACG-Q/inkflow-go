import sqlite3

conn = sqlite3.connect("C:/Users/LiuJi/Desktop/test4/inkflow-go/data/templates.db")
c = conn.cursor()
c.execute("SELECT name FROM sqlite_master WHERE type='table'")
tables = [r[0] for r in c.fetchall()]
print("Tables:", tables)

if "template_controls" in tables:
    c.execute("PRAGMA table_info(template_controls)")
    print("Columns:", [r[1] for r in c.fetchall()])
    c.execute("SELECT COUNT(*) FROM template_controls")
    print("Rows:", c.fetchone()[0])
