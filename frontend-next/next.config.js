/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
  images: {
    domains: ['realestate-pctr.c.yimg.jp'],
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**.yimg.jp',
      },
    ],
  },
}

module.exports = nextConfig
