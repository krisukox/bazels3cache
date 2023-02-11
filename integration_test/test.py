import subprocess, os, unittest


def print_cmd(cmd):
    print("+ " + " ".join(cmd))
    return cmd


class TestBazelCache(unittest.TestCase):
    def setUp(self):
        self.s3_host = os.getenv("s3_host", "localhost:9444")
        self.test_workspace = os.getenv("test_workspace", "workspace")
        self.bazels3cache = os.getenv("bazels3cache", "/bazels3cache")

    def start_bazels3cache(self):
        cmd = print_cmd(
            [
                self.bazels3cache,
                "--s3url",
                "http://{0}/s3".format(self.s3_host),
                "--bucket",
                "bazel",
            ]
        )
        subprocess.run(cmd, check=True)

    def stop_bazels3cache(self):
        cmd = print_cmd([self.bazels3cache, "--stop"])
        subprocess.run(cmd, check=True)

    def bazel_test(self):
        cmd = print_cmd(
            [
                "bazel",
                "test",
                "//...",
                "--remote_cache=http://localhost:7777",
                "--remote_upload_local_results=true",
            ]
        )
        result = subprocess.run(
            cmd,
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True,
            cwd=self.test_workspace,
        )
        print(result.stdout)
        return result.stdout

    def bazel_clean(self):
        cmd = print_cmd(["bazel", "clean"])
        result = subprocess.run(cmd, check=True, cwd=self.test_workspace)
        print(result.stdout)
        return result.stdout

    def test(self):
        self.start_bazels3cache()

        subprocess.run(
            [
                "bazel",
                "test",
                "//...",
                "--remote_cache=http://localhost:7777",
                "--remote_upload_local_results=true",
            ],
            check=True,
            cwd=self.test_workspace,
        )
        self.bazel_clean()
        stdout = self.bazel_test()

        self.assertNotEqual(stdout.find("12 remote cache hit"), -1)
        self.assertTrue(os.path.isfile(os.path.expanduser("~/.bazels3cache.log")))

        self.stop_bazels3cache()


if __name__ == "__main__":
    unittest.main()
