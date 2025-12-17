import { spawn } from "node:child_process";

const FRAMES = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];
const SPINNER_MS = 80;

const color = (code) => (msg) => `\x1b[${code}m${msg}\x1b[0m`;
const green = color(32);
const red = color(31);

const run = (label, cmd, args = []) =>
  new Promise((resolve, reject) => {
    let i = 0;
    process.stdout.write("\r" + green(`${FRAMES[0]} ${label}`));

    const spinner = setInterval(() => {
      process.stdout.write("\r" + green(`${FRAMES[++i % FRAMES.length]} ${label}`));
    }, SPINNER_MS);

    const child = spawn(cmd, args, { shell: true });

    child.stderr.on("data", (data) => process.stderr.write(data.toString()));

    child.on("exit", (code) => {
      clearInterval(spinner);
      if (code === 0) {
        process.stdout.write("\r" + green(`✅ ${label}\n`));
        resolve();
      } else {
        process.stdout.write("\r" + red(`❌ ${label}\n`));
        reject(new Error(`"${cmd} ${args.join(" ")}" exited with code ${code}`));
      }
    });
  });

const stashPush = () => run("Stashing uncommitted changes", "git", ["stash", "-k", "-u"]);
const stashPop = () => run("Restoring local changes", "git", ["stash", "pop"]);
const runTests = () => run("Running tests", "make", ["test"]);

export default {
  "**/*.go": async (files) => {
    if (!files?.length) return [];

    try {
      await stashPush();
      await runTests();
      await stashPop();
    } catch (err) {
      console.error(red(`🔄 ${err.message} — attempting to restore changes`));
      await stashPop();
      process.exit(1);
    }

    return [];
  },
};
