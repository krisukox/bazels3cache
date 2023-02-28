import subprocess, os, unittest, time, shutil, requests, sys


def print_cmd(cmd):
    print("\n+ " + " ".join(cmd))
    return cmd


class TestBazelCache(unittest.TestCase):
    def setUp(self):
        self.s3_host = os.getenv("S3_HOST", "localhost:9444")
        self.home_log_file = os.path.expanduser("~/.bazels3cache.log")
        self.bazels3cache = os.getenv("BAZELS3CACHE", "/bazels3cache")
        self.debug_env = os.getenv("DEBUG", "0")
        print(
            f"""
bazels3cache:       {self.bazels3cache}
debug:              {self.debug_env}
        """
        )

    def setup_test(self):
        self.test_workspace = os.getenv("TEST_WORKSPACE", "workspace")

    def setup_benchmark(self):
        self.delay_ms = os.getenv("delay_ms", "200")
        bazel_target = os.getenv("BAZEL_TARGET", "//src:bazel")
        self.bazel_target = bazel_target if bazel_target != "" else "//src:bazel"

        results_file = os.getenv("RESULTS_FILE", "results.txt")
        self.results_file = results_file if results_file != "" else "results.txt"

        log_file = os.getenv("LOG_FILE", ".bazels3cache.log")
        self.log_file = log_file if log_file != "" else ".bazels3cache.log"
        print(
            f"""
delay:              {self.delay_ms} ms
bazel target:       {self.bazel_target}
results file:       {self.results_file}
log file:           {self.log_file}
test:               {self._testMethodName}
        """
        )

    # run_cmd function runs commands, prints live stdout and returns
    # stdout as string
    def run_cmd(self, cmd, cwd=None, timeout=None):
        stdout = ""
        start_time = time.time()
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
            if timeout is not None and timeout < time.time() - start_time:
                process.kill()
                process.communicate()
                return stdout
        process.communicate()
        self.assertEqual(process.returncode, 0)
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
        # r = requests.get(url="http://localhost:7777/shutdown")

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

    def bazel_clean(self, cwd=None):
        return self.run_cmd(
            ["bazel", "clean"], cwd=self.test_workspace if cwd == None else cwd
        )

    def bazel_build(self, target, cwd=None):
        return self.run_cmd(
            [
                "bazel",
                "build",
                target,
                "--remote_cache=http://localhost:7777",
                "--remote_upload_local_results=true",
            ],
            cwd=cwd,
        )

    def simulate_delay(self):
        subprocess.run(
            [
                "tc",
                "qdisc",
                "add",
                "dev",
                "eth0",
                "root",
                "netem",
                "delay",
                self.delay_ms + "ms",
            ]
        )

    def performance_start(self):
        self.simulate_delay()

        self.start_bazels3cache()
        self.bazel_clean("/bazel")

    def performance_end(self, timeDelta):
        self.stop_bazels3cache()

        if self.debug_env == "1":
            with open(self.home_log_file, "r") as logfile:
                print(logfile.read())

        shutil.copyfile(self.home_log_file, "/results/" + self.log_file)

        print("Benchmark result: " + str(timeDelta) + " seconds.")

        results = open("/results/" + self.results_file, "a")
        results.writelines(str(timeDelta) + "\n")
        results.close()

    def test_integration(self):
        self.setup_test()
        self.start_bazels3cache()

        self.bazel_test()
        self.bazel_clean()
        stdout = self.bazel_test()

        self.assertNotEqual(stdout.find("12 remote cache hit"), -1)

        self.assertTrue(os.path.isfile(self.home_log_file))

        if self.debug_env == "1":
            with open(self.home_log_file, "r") as logfile:
                print(logfile.read())

        self.stop_bazels3cache()

    def test_performance_1(self):
        self.setup_benchmark()
        self.performance_start()

        start_time = time.time()

        self.bazel_build(self.bazel_target, "/bazel")

        timeDelta = time.time() - start_time

        self.performance_end(timeDelta)

    def test_performance_2(self):
        self.setup_benchmark()
        self.performance_start()

        start_time = time.time()

        self.bazel_build(self.bazel_target, "/bazel")
        self.bazel_clean("/bazel")
        self.bazel_build(self.bazel_target, "/bazel")

        timeDelta = time.time() - start_time

        self.performance_end(timeDelta)


if __name__ == "__main__":
    unittest.main()
