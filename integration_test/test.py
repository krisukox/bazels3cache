import subprocess, os, unittest, time


def print_cmd(cmd):
    print("+ " + " ".join(cmd))
    return cmd


class TestBazelCache(unittest.TestCase):
    def setUp(self):
        self.s3_host = os.getenv("s3_host", "localhost:9444")
        self.test_workspace = os.getenv("test_workspace", "workspace")
        self.bazels3cache = os.getenv("bazels3cache", "/bazels3cache")

    def run_cmd(self, cmd, cwd=None):
        stdout = ""
        process = subprocess.Popen(
            print_cmd(cmd),
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            universal_newlines=True,
            cwd=cwd,
        )
        while True:
            stdout_line = process.stdout.readline()
            if stdout_line == "" and process.poll() is not None:
                break
            if stdout_line:
                stdout += stdout_line
                print(stdout_line.strip())
        self.assertEquals(process.returncode, 0)
        return stdout

    def start_bazels3cache(self):
        self.run_cmd(
            [
                self.bazels3cache,
                "--s3url",
                "http://{0}/s3".format(self.s3_host),
                "--bucket",
                "bazel",
            ]
        )

    def stop_bazels3cache(self):
        self.run_cmd([self.bazels3cache, "--stop"])

    def bazel_test(self):
        return self.run_cmd(
            [
                "bazel",
                "test",
                "//...",
                "--remote_cache=http://localhost:7777",
                "--remote_upload_local_results=true",
            ],
            cwd=self.test_workspace,
        )

    def bazel_clean(self):
        return self.run_cmd(["bazel", "clean"], cwd=self.test_workspace)

    def test(self):
        self.start_bazels3cache()

        self.bazel_test()
        self.bazel_clean()
        stdout = self.bazel_test()

        self.assertNotEqual(stdout.find("12 remote cache hit"), -1)
        self.assertTrue(os.path.isfile(os.path.expanduser("~/.bazels3cache.log")))

        self.stop_bazels3cache()


if __name__ == "__main__":
    unittest.main()
