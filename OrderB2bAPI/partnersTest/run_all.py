"""Run all partner API tests in sequence. Run from project root or partnersTest dir."""
import os
import subprocess
import sys

script_dir = os.path.dirname(os.path.abspath(__file__))
os.chdir(script_dir)

scripts = [
    ("Health check", "test_health.py"),
    ("Auth failures", "test_auth_failures.py"),
    ("Admin list orders", "test_admin_list_orders.py"),
    ("Admin orders with filters", "test_admin_orders_with_filters.py"),
    ("Cart submit", "test_cart_submit.py"),
    ("Cart validation (expect 422)", "test_cart_validation.py"),
]

for name, script in scripts:
    print("\n" + "=" * 60)
    print(f"  {name}")
    print("=" * 60)
    subprocess.run([sys.executable, script])
