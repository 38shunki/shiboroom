import Link from 'next/link'

export default function PrivacyPage() {
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
        <h1 className="text-4xl font-bold mb-8">プライバシーポリシー</h1>

        <div className="bg-white rounded-lg shadow-sm p-8 space-y-8">
          <section>
            <p className="text-gray-700 leading-relaxed">
              shiboroom（以下「本サービス」）は、利用者の個人情報を以下の方針に基づき、適切に取り扱います。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">1. 個人情報の定義</h2>
            <p className="text-gray-700">
              本ポリシーにおいて「個人情報」とは、個人情報保護法に定める情報を指し、
              氏名、メールアドレス、IPアドレス、その他特定の個人を識別できる情報を含みます。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">2. 取得する情報</h2>
            <p className="text-gray-700 mb-4">本サービスでは、以下の情報を取得する場合があります。</p>

            <div className="space-y-4 ml-4">
              <div>
                <h3 className="font-bold text-gray-900 mb-2">1. 利用環境に関する情報</h3>
                <ul className="list-disc list-inside space-y-1 text-gray-700 ml-4">
                  <li>IPアドレス</li>
                  <li>ブラウザの種類、OS、端末情報</li>
                  <li>アクセス日時、閲覧履歴、操作履歴</li>
                </ul>
              </div>

              <div>
                <h3 className="font-bold text-gray-900 mb-2">2. サービス利用に伴い取得される情報</h3>
                <ul className="list-disc list-inside space-y-1 text-gray-700 ml-4">
                  <li>表示・非表示にした物件情報</li>
                  <li>利用者が行った指摘・フィードバックの内容</li>
                </ul>
              </div>

              <div>
                <h3 className="font-bold text-gray-900 mb-2">3. お問い合わせ時に取得する情報</h3>
                <ul className="list-disc list-inside space-y-1 text-gray-700 ml-4">
                  <li>メールアドレス</li>
                  <li>お問い合わせ内容</li>
                </ul>
              </div>
            </div>

            <p className="text-gray-700 mt-4 bg-blue-50 p-4 rounded">
              ※本サービスは、原則として氏名、住所、電話番号等の入力を求めません。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">3. 利用目的</h2>
            <p className="text-gray-700 mb-4">取得した情報は、以下の目的の範囲内で利用します。</p>
            <ul className="list-disc list-inside space-y-2 text-gray-700 ml-4">
              <li>本サービスの提供・運営・改善</li>
              <li>利用状況の分析および品質向上</li>
              <li>不正利用、迷惑行為等の防止</li>
              <li>お問い合わせへの対応</li>
              <li>サービスに関する重要なお知らせの通知</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">4. 指摘・フィードバック情報の取り扱い</h2>
            <p className="text-gray-700 mb-4">
              利用者が行った指摘・フィードバックは、
              本サービスの改善や、他の利用者・情報提供元が誤りや違和感に気づくための参考情報として利用される場合があります。
            </p>
            <p className="text-gray-700">
              これらの情報は、個人を特定できない形で取り扱います。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">5. Cookie等の利用について</h2>
            <p className="text-gray-700 mb-4">
              本サービスでは、利便性向上および利用状況分析のため、Cookieおよび類似技術を使用する場合があります。
            </p>
            <p className="text-gray-700">
              利用者は、ブラウザの設定によりCookieの使用を制限または無効化することができます。
              ただし、その場合、一部機能が正しく動作しない可能性があります。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">6. 第三者提供について</h2>
            <p className="text-gray-700 mb-4">
              本サービスは、以下の場合を除き、個人情報を第三者に提供することはありません。
            </p>
            <ul className="list-disc list-inside space-y-2 text-gray-700 ml-4">
              <li>利用者本人の同意がある場合</li>
              <li>法令に基づき開示が求められた場合</li>
              <li>人の生命、身体または財産の保護のために必要がある場合</li>
            </ul>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">7. 外部サービスとの連携</h2>
            <p className="text-gray-700">
              本サービスでは、外部サイトや第三者サービスへのリンクを含む場合があります。
              これら外部サービスにおける個人情報の取り扱いについては、各サービスのプライバシーポリシーをご確認ください。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">8. 安全管理措置</h2>
            <p className="text-gray-700">
              本サービスは、個人情報への不正アクセス、漏洩、改ざん、紛失等を防止するため、
              合理的かつ適切な安全管理措置を講じます。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">9. 個人情報の開示・訂正・削除</h2>
            <p className="text-gray-700">
              利用者は、自己の個人情報について、開示・訂正・削除等を求めることができます。
              希望される場合は、運営者までご連絡ください。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">10. プライバシーポリシーの変更</h2>
            <p className="text-gray-700">
              本ポリシーの内容は、法令の改正やサービス内容の変更等に応じて、予告なく変更されることがあります。
              変更後のプライバシーポリシーは、本サービス上に掲載した時点で効力を生じます。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">11. お問い合わせ窓口</h2>
            <p className="text-gray-700">
              本ポリシーに関するお問い合わせがある場合は、運営者（shiboroom 運営）までご連絡ください。
            </p>
          </section>

          <section className="pt-8 border-t border-gray-200">
            <p className="text-gray-600">制定日：2025年12月19日</p>
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
