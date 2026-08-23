import sqlite3

conn = sqlite3.connect("C:/Users/LiuJi/Desktop/test4/inkflow-go/data/templates.db")
c = conn.cursor()

# Check all controls
c.execute("SELECT id, template_id, label FROM template_controls")
print("All controls:")
for r in c.fetchall():
    print(f"  id={r[0]}, template_id={r[1]}, label={r[2]}")

# Check templates
c.execute("SELECT id, name FROM templates WHERE deleted_at IS NULL")
print("\nTemplates:")
for r in c.fetchall():
    print(f"  id={r[0]}, name={r[1]}")
