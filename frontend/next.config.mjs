/** @type {import('next').NextConfig} */

// Use repo name as basePath when deploying to GitHub Pages via Actions
const isGithubActions = process.env.GITHUB_ACTIONS || false;
let basePath = '';

if (isGithubActions) {
  const repo = process.env.GITHUB_REPOSITORY?.replace(/.*?\//, '') || 'PD-Hunter';
  basePath = `/${repo}`;
}

const nextConfig = {
  output: "export",
  images: {
    unoptimized: true,
  },
  basePath: basePath,
};

export default nextConfig;
