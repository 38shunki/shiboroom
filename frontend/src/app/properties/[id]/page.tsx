'use client'

import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8084'

interface Property {
  id: string
  detail_url: string
  title: string
  image_url?: string
  rent?: number
  floor_plan?: string
  floor_plan_details?: string
  area?: number
  walk_time?: number
  station?: string
  address?: string
  building_age?: number
  building_name?: string
  building_type?: string
  structure?: string
  direction?: string
  floor?: number
  floor_label?: string
  contract_period?: string
  insurance?: string
  parking?: string
  facilities?: string
  features?: string
  room_layout_image_url?: string
  management_fee?: string
  deposit?: string
  key_money?: string
  guarantor_deposit?: string
  security_deposit?: string
  move_in_date?: string
  conditions?: string
  notes?: string
  status: string
  fetched_at: string
  created_at: string
  updated_at: string
  removed_at?: string
}

interface PropertyStation {
  id: number
  property_id: string
  station_name: string
  line_name: string
  walk_minutes: number
  sort_order: number
  created_at: string
  updated_at: string
}

interface PropertySnapshot {
  id: number
  property_id: string
  snapshot_at: string
  rent?: number
  floor_plan?: string
  area?: number
  walk_time?: number
  station?: string
  address?: string
  building_age?: number
  floor?: number
  image_url?: string
  status: string
  has_changed: boolean
  change_note?: string
  created_at: string
}

interface PropertyChange {
  id: number
  property_id: string
  snapshot_id: number
  change_type: string
  old_value?: string
  new_value?: string
  change_magnitude?: number
  detected_at: string
}

export default function PropertyDetailPage() {
  const params = useParams()
  const router = useRouter()
  const propertyId = params.id as string

  const [property, setProperty] = useState<Property | null>(null)
  const [stations, setStations] = useState<PropertyStation[]>([])
  const [snapshots, setSnapshots] = useState<PropertySnapshot[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'overview' | 'history'>('overview')

  useEffect(() => {
    if (propertyId) {
      fetchPropertyDetails()
    }
  }, [propertyId])

  const fetchPropertyDetails = async () => {
    setLoading(true)
    setError(null)

    try {
      // Fetch property details
      const propertyResponse = await fetch(`${API_URL}/api/properties/${propertyId}`)
      if (!propertyResponse.ok) {
        throw new Error('物件が見つかりませんでした')
      }
      const responseData = await propertyResponse.json()

      // Handle new response format: {property, stations}
      if (responseData.property) {
        setProperty(responseData.property)
        setStations(responseData.stations || [])
      } else {
        // Fallback for old format (direct property object)
        setProperty(responseData)
        setStations([])
      }

      // Fetch snapshot history
      const historyResponse = await fetch(`${API_URL}/api/properties/${propertyId}/history?limit=30`)
      if (historyResponse.ok) {
        const historyData = await historyResponse.json()
        setSnapshots(historyData.snapshots || [])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '読み込みエラーが発生しました')
    } finally {
      setLoading(false)
    }
  }

  const formatRent = (rent?: number) => {
    if (!rent) return null
    return (rent / 10000).toFixed(1)
  }

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleDateString('ja-JP', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    })
  }

  const getChangeTypeLabel = (changeType: string) => {
    const labels: Record<string, string> = {
      'new_property': '新規登録',
      'rent_changed': '賃料変更',
      'status_changed': 'ステータス変更',
      'floor_plan_changed': '間取り変更',
      'area_changed': '面積変更',
      'building_age_changed': '築年数変更',
      'image_changed': '画像変更',
      'property_removed': '物件削除'
    }
    return labels[changeType] || changeType
  }

  const translateFacility = (facility: string): string => {
    const translations: Record<string, string> = {
      // 設備（英語フルネーム）
      'air_conditioner': 'エアコン',
      'auto_lock': 'オートロック',
      'balcony': 'バルコニー',
      'bath_toilet_separate': 'バス・トイレ別',
      'bathroom_dryer': '浴室乾燥機',
      'flooring': 'フローリング',
      'laundry_space': '室内洗濯機置場',
      'second_floor_plus': '2階以上',
      'washbasin': '洗面所独立',
      'independent_washbasin': '洗面所独立',
      'system_kitchen': 'システムキッチン',
      'corner_room': '角部屋',
      'delivery_box': '宅配ボックス',
      'bike_parking': '駐輪場',
      'elevator': 'エレベーター',
      'walk_in_closet': 'ウォークインクローゼット',
      'shower_toilet': '温水洗浄便座',
      'tile_flooring': 'タイル張り',
      'gas_stove': 'ガスコンロ対応',
      'shoe_box': 'シューズボックス',
      'intercom': 'TVモニタ付インターホン',
      'security_camera': '防犯カメラ',
      'indoor_washroom': '室内洗濯機置場',
      'loft': 'ロフト',
      'garden': '専用庭',
      'parking': '駐車場',
      'optical_fiber': '光ファイバー',
      'broadband': 'インターネット対応',
      'catv': 'CATV',
      'pet_friendly': 'ペット相談',
      'south_facing': '南向き',
      'two_family': '二人入居可',
      'storage': 'トランクルーム',
      'guarantor_unnecessary': '保証人不要',
      'deposit_free': '敷金なし',
      'key_money_free': '礼金なし',
      // 特徴（2文字コード）
      'ac': 'エアコン',
      'al': 'オートロック',
      'bc': 'バルコニー',
      'bd': '浴室乾燥機',
      'bt': 'バス・トイレ別',
      'bs': 'BS',
      'cs': 'CS',
      'ct': 'CATVインターネット',
      'cr': '角部屋',
      'db': '宅配ボックス',
      'em': 'エレベーター',
      'ep': '駐輪場',
      'ev': 'エレベーター',
      'fl': 'フローリング',
      'gr': 'ガスコンロ',
      'mp': '駐車場',
      'sf': '2階以上',
      'sk': 'システムキッチン',
      'tr': 'トランクルーム',
      'tw': '二人入居可',
      'vb': 'TVモニタ付インターホン',
      'wc': 'ウォークインクローゼット',
      'wg': '温水洗浄便座',
      'wl': '室内洗濯機置場',
      'wm': '洗面所独立',
      'cl': 'クローゼット',
      'ff': 'フリーレント',
      'fr': '冷蔵庫',
      'ma': '管理人',
      'nt': 'インターネット',
      'rm': 'リフォーム済',
      'sn': '新築',
      'so': '南向き',
      'tm': 'タイル',
      'ws': '洗濯機',
      'ih': 'IHコンロ',
      'le': 'LED照明',
      'lf': 'ロフト',
      'sc': '宅配ボックス',
      'tf': '都市ガス',
      'bn': 'BS/CS対応',
      'hb': '高速インターネット',
      'it': 'インターネット対応'
    }
    return translations[facility] || facility
  }

  const decodeUnicodeEscapes = (str: string): string => {
    // Decode Unicode escape sequences like \u968E to proper characters
    try {
      return str.replace(/\\u([\dA-Fa-f]{4})/g, (match, grp) => {
        return String.fromCharCode(parseInt(grp, 16))
      }).replace(/\\\//g, '/')  // Also decode \/ to /
    } catch {
      return str
    }
  }

  const parseFacilitiesString = (facilitiesStr: string): string[] => {
    try {
      // Handle JSON array format
      const parsed = JSON.parse(facilitiesStr)
      if (Array.isArray(parsed)) {
        return parsed
      }
      return [facilitiesStr]
    } catch {
      // Handle plain string or " / " separated format
      if (facilitiesStr.includes(' / ')) {
        return facilitiesStr.split(' / ')
      }
      return [facilitiesStr]
    }
  }

  if (loading) {
    return (
      <div className="container" style={{ padding: '40px 20px', textAlign: 'center' }}>
        <div className="loading"></div>
        <p style={{ marginTop: '20px', color: 'var(--text-muted)' }}>読み込み中...</p>
      </div>
    )
  }

  if (error || !property) {
    return (
      <div className="container" style={{ padding: '40px 20px', textAlign: 'center' }}>
        <p style={{ color: 'var(--text-muted)' }}>{error || '物件が見つかりませんでした'}</p>
        <Link href="/" style={{ marginTop: '20px', display: 'inline-block' }}>
          一覧に戻る
        </Link>
      </div>
    )
  }

  return (
    <div className="property-detail-page">
      <header className="detail-header">
        <div className="header-top">
          <Link href="/" className="back-button">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5">
              <path d="m15 18-6-6 6-6"/>
            </svg>
            <span>一覧に戻る</span>
          </Link>
        </div>
        <nav className="tab-nav">
          <button
            className={`tab-btn ${activeTab === 'overview' ? 'active' : ''}`}
            onClick={() => setActiveTab('overview')}
          >
            物件詳細
          </button>
          <button
            className={`tab-btn ${activeTab === 'history' ? 'active' : ''}`}
            onClick={() => setActiveTab('history')}
          >
            変更履歴
            {snapshots.length > 0 && <span className="tab-count">{snapshots.length}</span>}
          </button>
        </nav>
      </header>

      <main className="detail-content">
        {activeTab === 'overview' && (
          <div className="overview-section">
            <div className="property-hero">
              {property.image_url && (
                <div className="property-hero-image">
                  <img src={property.image_url} alt={property.title} />
                </div>
              )}
              <div className="property-hero-content">
                <h1 className="property-title">{property.title}</h1>
                {property.rent && (
                  <div className="property-rent-large">
                    {formatRent(property.rent)}<span>万円</span>
                  </div>
                )}
              </div>
            </div>

            <div className="property-details-grid">
              <div className="detail-card">
                <h3 className="detail-card-title">基本情報</h3>
                <dl className="detail-list">
                  {property.building_name && (
                    <>
                      <dt>建物名</dt>
                      <dd>{property.building_name}</dd>
                    </>
                  )}
                  {property.floor_plan && (
                    <>
                      <dt>間取り</dt>
                      <dd>{property.floor_plan}</dd>
                    </>
                  )}
                  {property.floor_plan_details && (
                    <>
                      <dt>間取り詳細</dt>
                      <dd>{property.floor_plan_details}</dd>
                    </>
                  )}
                  {property.area && (
                    <>
                      <dt>専有面積</dt>
                      <dd>{property.area}㎡</dd>
                    </>
                  )}
                  {property.direction && (
                    <>
                      <dt>向き</dt>
                      <dd>{property.direction}</dd>
                    </>
                  )}
                </dl>
              </div>

              <div className="detail-card">
                <h3 className="detail-card-title">建物情報</h3>
                <dl className="detail-list">
                  {property.building_age !== undefined && (
                    <>
                      <dt>築年数</dt>
                      <dd>築{property.building_age}年</dd>
                    </>
                  )}
                  {property.structure && (
                    <>
                      <dt>構造</dt>
                      <dd>{property.structure}</dd>
                    </>
                  )}
                  {property.floor_label && (
                    <>
                      <dt>階</dt>
                      <dd>{decodeUnicodeEscapes(property.floor_label)}</dd>
                    </>
                  )}
                  {property.floor && !property.floor_label && (
                    <>
                      <dt>階数</dt>
                      <dd>{property.floor}階</dd>
                    </>
                  )}
                </dl>
              </div>

              <div className="detail-card">
                <h3 className="detail-card-title">アクセス{stations.length > 0 && `（${stations.length}駅）`}</h3>
                <dl className="detail-list">
                  {stations.length > 0 ? (
                    <>
                      <dt>最寄駅</dt>
                      <dd>
                        <div className="stations-list">
                          {stations.map((station, index) => (
                            <div key={station.id} className="station-item">
                              {index === 0 && <span className="nearest-badge">最寄り</span>}
                              <span className="station-name">{station.station_name}</span>
                              <span className="station-line">{station.line_name}</span>
                              {station.walk_minutes !== null && station.walk_minutes !== undefined ? (
                                <span className="station-walk">徒歩{station.walk_minutes}分</span>
                              ) : (
                                <span className="station-walk">徒歩不明</span>
                              )}
                            </div>
                          ))}
                        </div>
                      </dd>
                    </>
                  ) : property.station && property.walk_time ? (
                    <>
                      <dt>最寄駅</dt>
                      <dd>{property.station} 徒歩{property.walk_time}分</dd>
                    </>
                  ) : null}
                  {property.address && (
                    <>
                      <dt>所在地</dt>
                      <dd>{property.address}</dd>
                    </>
                  )}
                </dl>
              </div>

              <div className="detail-card">
                <h3 className="detail-card-title">費用詳細</h3>
                <dl className="detail-list">
                  {property.rent && (
                    <>
                      <dt>賃料</dt>
                      <dd className="highlight-value">{formatRent(property.rent)}万円</dd>
                    </>
                  )}
                  {property.management_fee && (
                    <>
                      <dt>管理費・共益費</dt>
                      <dd>{property.management_fee}</dd>
                    </>
                  )}
                  {property.deposit && (
                    <>
                      <dt>敷金</dt>
                      <dd>{property.deposit}</dd>
                    </>
                  )}
                  {property.key_money && (
                    <>
                      <dt>礼金</dt>
                      <dd>{property.key_money}</dd>
                    </>
                  )}
                  {property.guarantor_deposit && (
                    <>
                      <dt>保証金</dt>
                      <dd>{property.guarantor_deposit}</dd>
                    </>
                  )}
                  {property.security_deposit && (
                    <>
                      <dt>敷引</dt>
                      <dd>{property.security_deposit}</dd>
                    </>
                  )}
                </dl>
              </div>

              <div className="detail-card">
                <h3 className="detail-card-title">契約情報</h3>
                <dl className="detail-list">
                  {property.move_in_date && (
                    <>
                      <dt>入居可能日</dt>
                      <dd>{property.move_in_date}</dd>
                    </>
                  )}
                  {property.contract_period && (
                    <>
                      <dt>契約期間</dt>
                      <dd>{property.contract_period}</dd>
                    </>
                  )}
                  {property.insurance && (
                    <>
                      <dt>保険</dt>
                      <dd>{property.insurance}</dd>
                    </>
                  )}
                  {property.parking && (
                    <>
                      <dt>駐車場</dt>
                      <dd>{property.parking}</dd>
                    </>
                  )}
                  {property.conditions && (
                    <>
                      <dt>入居条件</dt>
                      <dd>{property.conditions}</dd>
                    </>
                  )}
                </dl>
              </div>

              <div className="detail-card">
                <h3 className="detail-card-title">データ情報</h3>
                <dl className="detail-list">
                  <dt>登録日時</dt>
                  <dd>{formatDate(property.created_at)}</dd>
                  <dt>最終更新</dt>
                  <dd>{formatDate(property.updated_at)}</dd>
                  <dt>取得日時</dt>
                  <dd>{formatDate(property.fetched_at)}</dd>
                </dl>
              </div>
            </div>

            {(property.facilities || property.features) && (
              <div className="detail-card-full">
                <h3 className="detail-card-title">設備・特徴</h3>
                <div className="facilities-grid">
                  {property.facilities && (
                    <div className="facility-section">
                      <h4 className="facility-section-title">設備</h4>
                      <div className="facility-tags">
                        {parseFacilitiesString(property.facilities).map((facility, index) => (
                          <span key={index} className="facility-tag">{translateFacility(facility)}</span>
                        ))}
                      </div>
                    </div>
                  )}
                  {property.features && (
                    <div className="facility-section">
                      <h4 className="facility-section-title">特徴</h4>
                      <div className="facility-tags">
                        {parseFacilitiesString(property.features).map((feature, index) => (
                          <span key={index} className="facility-tag feature">{translateFacility(feature)}</span>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}

            {property.room_layout_image_url && (
              <div className="detail-card-full">
                <h3 className="detail-card-title">間取り図</h3>
                <div className="layout-image-wrapper">
                  <img src={property.room_layout_image_url} alt="間取り図" />
                </div>
              </div>
            )}

            {property.notes && (
              <div className="detail-card-full">
                <h3 className="detail-card-title">備考</h3>
                <div className="notes-content">
                  {property.notes}
                </div>
              </div>
            )}

            <div className="detail-actions">
              <a
                href={property.detail_url}
                className="btn btn-primary btn-large"
                target="_blank"
                rel="noreferrer"
              >
                Yahoo!不動産で詳細を見る
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                  <polyline points="15 3 21 3 21 9"/>
                  <line x1="10" x2="21" y1="14" y2="3"/>
                </svg>
              </a>
            </div>
          </div>
        )}

        {activeTab === 'history' && (
          <div className="history-section">
            <div className="history-header">
              <h2>変更履歴</h2>
              <p className="history-description">
                この物件のスナップショット履歴と検出された変更を表示しています。
              </p>
            </div>

            {snapshots.length === 0 ? (
              <div className="empty-state">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M3 3v18h18"/>
                  <path d="m19 9-5 5-4-4-3 3"/>
                </svg>
                <p>まだ履歴データがありません</p>
              </div>
            ) : (
              <div className="timeline">
                {snapshots.map((snapshot, index) => (
                  <div key={snapshot.id} className="timeline-item">
                    <div className="timeline-marker">
                      {snapshot.has_changed && (
                        <div className="timeline-badge">
                          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                            <path d="M12 20h9"/>
                            <path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/>
                          </svg>
                        </div>
                      )}
                    </div>
                    <div className="timeline-content">
                      <div className="snapshot-card">
                        <div className="snapshot-header">
                          <div className="snapshot-date">
                            {formatDate(snapshot.snapshot_at)}
                          </div>
                          {snapshot.has_changed && (
                            <div className="snapshot-badge changed">変更あり</div>
                          )}
                        </div>
                        <div className="snapshot-body">
                          <div className="snapshot-info">
                            {snapshot.rent && (
                              <div className="info-item">
                                <span className="info-label">賃料</span>
                                <span className="info-value highlight">
                                  {formatRent(snapshot.rent)}万円
                                </span>
                              </div>
                            )}
                            {snapshot.floor_plan && (
                              <div className="info-item">
                                <span className="info-label">間取り</span>
                                <span className="info-value">{snapshot.floor_plan}</span>
                              </div>
                            )}
                            {snapshot.area && (
                              <div className="info-item">
                                <span className="info-label">面積</span>
                                <span className="info-value">{snapshot.area}㎡</span>
                              </div>
                            )}
                            {snapshot.building_age !== undefined && (
                              <div className="info-item">
                                <span className="info-label">築年数</span>
                                <span className="info-value">築{snapshot.building_age}年</span>
                              </div>
                            )}
                          </div>
                          {snapshot.change_note && (
                            <div className="change-note">
                              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                                <circle cx="12" cy="12" r="10"/>
                                <path d="M12 16v-4"/>
                                <path d="M12 8h.01"/>
                              </svg>
                              {snapshot.change_note}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </main>

      <style jsx>{`
        .property-detail-page {
          min-height: 100vh;
          background: var(--bg-primary);
        }

        .detail-header {
          background: white;
          border-bottom: 1px solid var(--border-color);
          position: sticky;
          top: 0;
          z-index: 100;
          box-shadow: 0 1px 3px rgba(0,0,0,0.05);
          overflow: hidden;
        }

        .header-top {
          max-width: 1200px;
          margin: 0 auto;
          padding: 12px 20px;
          border-bottom: 1px solid var(--border-color);
          overflow: hidden;
        }

        .back-button {
          display: inline-flex;
          align-items: center;
          gap: 6px;
          padding: 8px 14px;
          background: transparent;
          border: 1px solid var(--border-color);
          border-radius: 8px;
          color: var(--text-primary);
          text-decoration: none;
          font-size: 14px;
          font-weight: 500;
          transition: all 0.2s;
        }

        .back-button:hover {
          background: var(--bg-secondary);
          border-color: var(--text-muted);
        }

        .back-button svg {
          flex-shrink: 0;
        }

        .tab-nav {
          max-width: 1200px;
          margin: 0 auto;
          padding: 0 20px;
          display: flex;
          gap: 4px;
        }

        .tab-btn {
          position: relative;
          padding: 14px 24px;
          background: none;
          border: none;
          font-size: 15px;
          font-weight: 600;
          color: var(--text-muted);
          cursor: pointer;
          transition: all 0.2s;
          border-bottom: 3px solid transparent;
        }

        .tab-btn:hover {
          color: var(--text-primary);
        }

        .tab-btn.active {
          color: var(--primary);
          border-bottom-color: var(--primary);
        }

        .tab-count {
          display: inline-block;
          margin-left: 6px;
          padding: 2px 8px;
          background: var(--primary);
          color: white;
          border-radius: 12px;
          font-size: 12px;
          font-weight: 600;
        }

        .tab-btn:not(.active) .tab-count {
          background: var(--bg-tertiary);
          color: var(--text-muted);
        }

        .detail-content {
          max-width: 1200px;
          margin: 0 auto;
          padding: 40px 20px;
        }

        .property-hero {
          background: white;
          border-radius: 12px;
          overflow: hidden;
          margin-bottom: 24px;
        }

        .property-hero-image {
          width: 100%;
          height: 400px;
          overflow: hidden;
        }

        .property-hero-image img {
          width: 100%;
          height: 100%;
          object-fit: cover;
        }

        .property-hero-content {
          padding: 32px;
        }

        .property-title {
          font-size: 24px;
          font-weight: 700;
          color: var(--text-primary);
          margin: 0 0 16px 0;
        }

        .property-rent-large {
          font-size: 36px;
          font-weight: 700;
          color: var(--primary);
        }

        .property-rent-large span {
          font-size: 18px;
          margin-left: 4px;
        }

        .property-details-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
          gap: 20px;
          margin-bottom: 24px;
        }

        .detail-card {
          background: white;
          border-radius: 12px;
          padding: 24px;
        }

        .detail-card-title {
          font-size: 16px;
          font-weight: 700;
          color: var(--text-primary);
          margin: 0 0 16px 0;
        }

        .detail-list {
          display: grid;
          grid-template-columns: 100px 1fr;
          gap: 12px;
          margin: 0;
        }

        .detail-list dt {
          font-size: 14px;
          color: var(--text-muted);
          font-weight: 500;
        }

        .detail-list dd {
          font-size: 14px;
          color: var(--text-primary);
          margin: 0;
        }

        .detail-list dd.highlight-value {
          color: var(--primary);
          font-weight: 600;
          font-size: 16px;
        }

        .stations-list {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }

        .station-item {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 8px;
          background: var(--bg-secondary);
          border-radius: 6px;
        }

        .station-name {
          font-weight: 600;
          color: var(--text-primary);
        }

        .station-line {
          font-size: 13px;
          color: var(--text-muted);
          flex: 1;
        }

        .station-walk {
          font-size: 13px;
          color: var(--primary);
          font-weight: 500;
        }

        .nearest-badge {
          display: inline-block;
          padding: 2px 8px;
          background: var(--primary);
          color: white;
          border-radius: 4px;
          font-size: 11px;
          font-weight: 600;
        }

        .detail-card-full {
          background: white;
          border-radius: 12px;
          padding: 24px;
          margin-bottom: 20px;
          grid-column: 1 / -1;
        }

        .facilities-grid {
          display: flex;
          flex-direction: column;
          gap: 20px;
        }

        .facility-section {
          display: flex;
          flex-direction: column;
          gap: 12px;
        }

        .facility-section-title {
          font-size: 14px;
          font-weight: 600;
          color: var(--text-primary);
          margin: 0;
        }

        .facility-tags {
          display: flex;
          flex-wrap: wrap;
          gap: 8px;
        }

        .facility-tag {
          padding: 6px 12px;
          background: var(--bg-secondary);
          color: var(--text-primary);
          border-radius: 6px;
          font-size: 13px;
          border: 1px solid var(--border-color);
        }

        .facility-tag.feature {
          background: var(--bg-tertiary);
          border-color: var(--primary);
          color: var(--primary);
        }

        .layout-image-wrapper {
          width: 100%;
          max-width: 600px;
          margin: 0 auto;
        }

        .layout-image-wrapper img {
          width: 100%;
          height: auto;
          border-radius: 8px;
          border: 1px solid var(--border-color);
        }

        .notes-content {
          font-size: 14px;
          line-height: 1.8;
          color: var(--text-primary);
          white-space: pre-wrap;
          padding: 16px;
          background: var(--bg-secondary);
          border-radius: 8px;
          border-left: 4px solid var(--primary);
        }

        .detail-actions {
          display: flex;
          justify-content: center;
          padding: 20px 0;
        }

        .btn-large {
          padding: 16px 32px;
          font-size: 16px;
          display: inline-flex;
          align-items: center;
          gap: 12px;
        }

        .btn-large svg {
          width: 20px;
          height: 20px;
        }

        .history-section {
          background: white;
          border-radius: 12px;
          padding: 32px;
        }

        .history-header {
          margin-bottom: 32px;
        }

        .history-header h2 {
          font-size: 24px;
          font-weight: 700;
          color: var(--text-primary);
          margin: 0 0 8px 0;
        }

        .history-description {
          font-size: 14px;
          color: var(--text-muted);
          margin: 0;
        }

        .timeline {
          position: relative;
          padding-left: 40px;
        }

        .timeline::before {
          content: '';
          position: absolute;
          left: 15px;
          top: 0;
          bottom: 0;
          width: 2px;
          background: var(--border-color);
        }

        .timeline-item {
          position: relative;
          padding-bottom: 32px;
        }

        .timeline-item:last-child {
          padding-bottom: 0;
        }

        .timeline-marker {
          position: absolute;
          left: -28px;
          top: 0;
          width: 32px;
          height: 32px;
          border-radius: 50%;
          background: white;
          border: 2px solid var(--border-color);
          display: flex;
          align-items: center;
          justify-content: center;
        }

        .timeline-badge {
          width: 20px;
          height: 20px;
          border-radius: 50%;
          background: var(--accent);
          display: flex;
          align-items: center;
          justify-content: center;
        }

        .timeline-badge svg {
          width: 12px;
          height: 12px;
          color: white;
        }

        .timeline-content {
          margin-left: 20px;
        }

        .snapshot-card {
          background: var(--bg-secondary);
          border-radius: 8px;
          overflow: hidden;
        }

        .snapshot-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 16px;
          border-bottom: 1px solid var(--border-color);
        }

        .snapshot-date {
          font-size: 14px;
          font-weight: 600;
          color: var(--text-primary);
        }

        .snapshot-badge {
          padding: 4px 12px;
          border-radius: 12px;
          font-size: 12px;
          font-weight: 600;
        }

        .snapshot-badge.changed {
          background: var(--accent);
          color: white;
        }

        .snapshot-body {
          padding: 16px;
        }

        .snapshot-info {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
          gap: 16px;
        }

        .info-item {
          display: flex;
          flex-direction: column;
          gap: 4px;
        }

        .info-label {
          font-size: 12px;
          color: var(--text-muted);
          font-weight: 500;
        }

        .info-value {
          font-size: 14px;
          color: var(--text-primary);
          font-weight: 600;
        }

        .info-value.highlight {
          color: var(--primary);
          font-size: 16px;
        }

        .change-note {
          margin-top: 16px;
          padding: 12px;
          background: white;
          border-radius: 6px;
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 13px;
          color: var(--text-primary);
        }

        .change-note svg {
          width: 16px;
          height: 16px;
          flex-shrink: 0;
          color: var(--accent);
        }

        .empty-state {
          text-align: center;
          padding: 60px 20px;
          color: var(--text-muted);
        }

        .empty-state svg {
          width: 64px;
          height: 64px;
          margin-bottom: 16px;
          opacity: 0.5;
        }

        .empty-state p {
          margin: 0;
          font-size: 16px;
        }

        .loading {
          width: 40px;
          height: 40px;
          border: 3px solid var(--border-color);
          border-top-color: var(--primary);
          border-radius: 50%;
          animation: spin 1s linear infinite;
          margin: 0 auto;
        }

        @keyframes spin {
          to { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  )
}
