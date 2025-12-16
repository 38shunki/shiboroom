import type { Metadata } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'shiboroom',
  description: '物件をふるい分けする',
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="ja">
      <body>{children}</body>
    </html>
  )
}
