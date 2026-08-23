import subprocess
import time
import sys
import os
import requests
import signal

SERVER_PORT = 9876
BASE_URL = f"http://localhost:{SERVER_PORT}"


def wait_for_server(timeout=15):
    start = time.time()
    while time.time() - start < timeout:
        try:
            r = requests.get(f"{BASE_URL}/api/health", timeout=2)
            if r.status_code == 200:
                return True
        except requests.ConnectionError:
            pass
        time.sleep(0.5)
    return False


def main():
    os.chdir(os.path.dirname(os.path.abspath(__file__)))
    server = subprocess.Popen(
        ["go", "run", "."],
        env={**os.environ, "PORT": str(SERVER_PORT), "LOG_LEVEL": "error"},
        cwd=os.path.join(os.path.dirname(__file__), ".."),
    )
    try:
        if not wait_for_server():
            print("ERROR: Server did not start")
            server.kill()
            sys.exit(1)
        print("Server started, running tests...")
        result = subprocess.run(["pytest", "-v", "--rootdir=."], capture_output=False)
        sys.exit(result.returncode)
    finally:
        server.terminate()
        server.wait()


if __name__ == "__main__":
    main()
