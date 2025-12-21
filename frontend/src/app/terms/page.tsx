import Link from 'next/link'

export default function TermsPage() {
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
        <h1 className="text-4xl font-bold mb-8">利用規約</h1>

        <div className="bg-white rounded-lg shadow-sm p-8 space-y-8">
          <section>
            <p className="text-gray-700 leading-relaxed">
              本利用規約（以下「本規約」）は、shiboroom（以下「本サービス」）が提供するサービスの利用条件を定めるものです。
              本サービスを利用するすべての利用者は、本規約に同意したものとみなされます。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第1条（サービスの目的）</h2>
            <ol className="list-decimal list-inside space-y-2 text-gray-700">
              <li>本サービスは、賃貸物件情報を整理・可視化し、不要な物件を除外することで、利用者の判断コストを低減することを目的とした物件整理支援サービスです。</li>
              <li>本サービスは、不動産の仲介、斡旋、契約行為を行うものではありません。</li>
            </ol>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第2条（定義）</h2>
            <p className="text-gray-700 mb-4">本規約において、以下の用語はそれぞれ次の意味を有します。</p>
            <dl className="space-y-4">
              <div>
                <dt className="font-bold text-gray-900">利用者</dt>
                <dd className="ml-4 text-gray-700">本サービスを閲覧または利用するすべての個人または法人。</dd>
              </div>
              <div>
                <dt className="font-bold text-gray-900">物件情報</dt>
                <dd className="ml-4 text-gray-700">本サービス上に表示される、賃貸物件に関する情報全般。</dd>
              </div>
              <div>
                <dt className="font-bold text-gray-900">指摘・フィードバック</dt>
                <dd className="ml-4 text-gray-700">利用者が物件情報に関して行う、誤り・違和感・不正確な点等に関する報告や意見。</dd>
              </div>
            </dl>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第3条（サービス内容）</h2>
            <ol className="list-decimal list-inside space-y-2 text-gray-700 mb-4">
              <li>本サービスは、以下の機能を提供します。
                <ul className="list-disc list-inside ml-6 mt-2 space-y-1">
                  <li>賃貸物件情報の表示・整理</li>
                  <li>不要な物件を非表示にする機能</li>
                  <li>利用者による指摘・フィードバック機能</li>
                  <li>その他、これらに付随する機能</li>
                </ul>
              </li>
              <li>運営者は、サービス内容を予告なく変更、追加、削除することがあります。</li>
            </ol>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第4条（情報の正確性に関する考え方）</h2>
            <ol className="list-decimal list-inside space-y-2 text-gray-700">
              <li>本サービスに掲載される物件情報は、常に正確・完全・最新であることを保証するものではありません。</li>
              <li>本サービスは、利用者の指摘やフィードバックを通じて、他の利用者や情報提供元が誤りや違和感に気づく「きっかけ」を提供することを理念としています。</li>
              <li>利用者は、内見、申込、契約等の判断を行う際、必ず公式の掲載元、管理会社、貸主等に直接確認するものとします。</li>
            </ol>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第5条（利用者の責任）</h2>
            <ol className="list-decimal list-inside space-y-2 text-gray-700">
              <li>利用者は、自らの判断と責任において本サービスを利用するものとします。</li>
              <li>本サービス上の情報のみを根拠として、契約その他の法的判断を行わないものとします。</li>
              <li>利用者が行った指摘・フィードバックについては、利用者自身が責任を負うものとします。</li>
            </ol>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第6条（禁止事項）</h2>
            <p className="text-gray-700 mb-4">利用者は、以下の行為を行ってはなりません。</p>
            <ol className="list-decimal list-inside space-y-2 text-gray-700">
              <li>法令または公序良俗に違反する行為</li>
              <li>虚偽、誤解を招く、または悪意のある指摘を行う行為</li>
              <li>本サービスの運営を妨害する行為</li>
              <li>他の利用者、第三者、または運営者の権利・利益を侵害する行為</li>
              <li>不正アクセス、過度な負荷を与える行為</li>
              <li>その他、運営者が不適切と判断する行為</li>
            </ol>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第7条（知的財産権）</h2>
            <ol className="list-decimal list-inside space-y-2 text-gray-700">
              <li>本サービスに関するUI、デザイン、構成、文章、プログラム等の知的財産権は、運営者または正当な権利者に帰属します。</li>
              <li>利用者は、運営者の許可なく、これらを複製、転載、改変、再配布してはなりません。</li>
            </ol>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第8条（免責事項）</h2>
            <ol className="list-decimal list-inside space-y-2 text-gray-700">
              <li>本サービスの利用により生じた損害について、運営者は一切の責任を負いません。</li>
              <li>物件情報の誤り、変更、掲載停止等により生じた不利益についても、運営者は責任を負いません。</li>
              <li>外部サイトや第三者サービスに起因するトラブルについて、運営者は責任を負いません。</li>
            </ol>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第9条（サービスの中断・終了）</h2>
            <p className="text-gray-700 mb-4">運営者は、以下の場合、事前の通知なく本サービスの全部または一部を中断・終了することがあります。</p>
            <ol className="list-decimal list-inside space-y-2 text-gray-700">
              <li>システム保守、障害対応の場合</li>
              <li>天災、通信障害等の不可抗力による場合</li>
              <li>その他、運営者が必要と判断した場合</li>
            </ol>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第10条（規約の変更）</h2>
            <p className="text-gray-700">
              運営者は、必要に応じて本規約を変更することがあります。
              変更後の規約は、本サービス上に掲載した時点で効力を生じるものとします。
            </p>
          </section>

          <section>
            <h2 className="text-2xl font-bold mb-4">第11条（準拠法および管轄）</h2>
            <p className="text-gray-700">
              本規約は、日本法を準拠法とします。
              本サービスに関して生じた紛争については、運営者所在地を管轄する裁判所を専属的合意管轄とします。
            </p>
          </section>

          <section className="pt-8 border-t border-gray-200">
            <p className="text-gray-600 text-right">以上</p>
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
