/** @type {import('next').NextConfig} */
module.exports = {
  reactStrictMode: true,
  // Emit a self-contained server at .next/standalone so the Docker
  // runtime image only needs Node + the minimal traced node_modules.
  output: "standalone",
};
