import Link from 'next/link'

export default function TokushohoPage() {
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white shadow-sm">
        <div className="max-w-4xl mx-auto px-4 py-6">
          <Link href="/" className="text-blue-600 hover:text-blue-800">
            ← トップページに戻る
          </Link>
        </div>
      </header>

      <main className="max-w-4xl mx-auto px-4 py-12">
        <h1 className="text-4xl font-bold mb-8">特定商取引法に基づく表記</h1>

        <div className="bg-white rounded-lg shadow-sm p-8 space-y-8">
          <section>
            <p className="text-gray-700 leading-relaxed">
              本サービス「shiboroom」は、無料で提供されており、現時点では特定商取引法の対象となる販売行為は行っておりません。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">サービスの性質</h2>
            <ul className="list-disc list-inside space-y-2 text-gray-700 ml-4">
              <li>本サービスは、賃貸物件情報の整理・可視化を支援する無料サービスです</li>
              <li>不動産の仲介、斡旋、契約代行は行いません</li>
              <li>利用者から料金を徴収することはありません</li>
              <li>広告掲載等の有料サービスは現時点では提供しておりません</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">運営者情報</h2>
            <dl className="space-y-4">
              <div className="border-b border-gray-200 pb-4">
                <dt className="font-bold text-gray-900 mb-2">サービス名</dt>
                <dd className="ml-4 text-gray-700">shiboroom（しぼるーむ）</dd>
              </div>
              <div className="border-b border-gray-200 pb-4">
                <dt className="font-bold text-gray-900 mb-2">運営者</dt>
                <dd className="ml-4 text-gray-700">個人運営</dd>
              </div>
              <div className="border-b border-gray-200 pb-4">
                <dt className="font-bold text-gray-900 mb-2">サービス提供地域</dt>
                <dd className="ml-4 text-gray-700">日本国内</dd>
              </div>
              <div className="border-b border-gray-200 pb-4">
                <dt className="font-bold text-gray-900 mb-2">お問い合わせ</dt>
                <dd className="ml-4 text-gray-700">運営者までご連絡ください</dd>
              </div>
            </dl>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">免責事項</h2>
            <ul className="list-disc list-inside space-y-2 text-gray-700 ml-4">
              <li>本サービスは情報提供を目的としており、不動産取引の仲介・斡旋は行いません</li>
              <li>物件情報の正確性、完全性、最新性を保証するものではありません</li>
              <li>本サービスの利用により生じたいかなる損害についても、運営者は責任を負いません</li>
              <li>外部サイトへのリンクは参考情報として提供しており、その内容について責任を負いません</li>
            </ul>
          </section>

          <section className="bg-blue-50 p-6 rounded-lg">
            <h2 className="text-xl font-bold mb-4 text-blue-900">将来的な有料サービスについて</h2>
            <p className="text-gray-700 mb-4">
              将来的に有料プランやプレミアム機能を提供する場合には、本ページを更新し、特定商取引法に基づく必要な情報を掲載いたします。
            </p>
            <p className="text-gray-700">
              有料化の際には、事前に利用者へ十分な告知を行い、同意を得た上でサービスを提供いたします。
            </p>
          </section>

          <section className="pt-8 border-t border-gray-200">
            <p className="text-gray-600">制定日：2025年12月18日</p>
            <p className="text-gray-600 text-right mt-4">以上</p>
          </section>
        </div>

        <div className="mt-8 text-center">
          <Link
            href="/"
            className="inline-block bg-blue-600 text-white px-8 py-3 rounded-lg hover:bg-blue-700 transition-colors"
          >
            トップページに戻る
          </Link>
        </div>
      </main>

      <footer className="bg-white border-t border-gray-200 mt-16">
        <div className="max-w-4xl mx-auto px-4 py-8 text-center text-gray-600">
          <div className="space-x-6">
            <Link href="/terms" className="hover:text-gray-900">利用規約</Link>
            <Link href="/privacy" className="hover:text-gray-900">プライバシーポリシー</Link>
            <Link href="/tokushoho" className="hover:text-gray-900">特定商取引法に基づく表記</Link>
          </div>
          <p className="mt-4 text-sm">© 2025 shiboroom</p>
        </div>
      </footer>
    </div>
  )
}
