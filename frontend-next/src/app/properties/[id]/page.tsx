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
  area?: number
  walk_time?: number
  station?: string
  address?: string
  building_age?: number
  floor?: number
  status: string
  fetched_at: string
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
      const propertyData = await propertyResponse.json()
      setProperty(propertyData)

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
        <div className="header-inner">
          <Link href="/" className="back-button">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="m15 18-6-6 6-6"/>
            </svg>
            一覧に戻る
          </Link>
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
              変更履歴 {snapshots.length > 0 && `(${snapshots.length})`}
            </button>
          </nav>
        </div>
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
                  {property.floor_plan && (
                    <>
                      <dt>間取り</dt>
                      <dd>{property.floor_plan}</dd>
                    </>
                  )}
                  {property.area && (
                    <>
                      <dt>専有面積</dt>
                      <dd>{property.area}㎡</dd>
                    </>
                  )}
                  {property.building_age !== undefined && (
                    <>
                      <dt>築年数</dt>
                      <dd>築{property.building_age}年</dd>
                    </>
                  )}
                  {property.floor && (
                    <>
                      <dt>階数</dt>
                      <dd>{property.floor}階</dd>
                    </>
                  )}
                </dl>
              </div>

              <div className="detail-card">
                <h3 className="detail-card-title">アクセス・所在地</h3>
                <dl className="detail-list">
                  {property.station && property.walk_time && (
                    <>
                      <dt>最寄駅</dt>
                      <dd>{property.station} 徒歩{property.walk_time}分</dd>
                    </>
                  )}
                  {property.address && (
                    <>
                      <dt>所在地</dt>
                      <dd>{property.address}</dd>
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
        }

        .header-inner {
          max-width: 1200px;
          margin: 0 auto;
          padding: 16px 20px;
          display: flex;
          align-items: center;
          justify-content: space-between;
          gap: 20px;
        }

        .back-button {
          display: flex;
          align-items: center;
          gap: 8px;
          padding: 8px 16px;
          background: var(--bg-secondary);
          border-radius: 8px;
          color: var(--text-primary);
          text-decoration: none;
          font-size: 14px;
          font-weight: 500;
          transition: background-color 0.2s;
        }

        .back-button:hover {
          background: var(--bg-tertiary);
        }

        .back-button svg {
          width: 16px;
          height: 16px;
        }

        .tab-nav {
          display: flex;
          gap: 8px;
        }

        .tab-btn {
          padding: 10px 20px;
          background: none;
          border: none;
          border-radius: 8px;
          font-size: 14px;
          font-weight: 500;
          color: var(--text-muted);
          cursor: pointer;
          transition: all 0.2s;
        }

        .tab-btn:hover {
          background: var(--bg-secondary);
          color: var(--text-primary);
        }

        .tab-btn.active {
          background: var(--primary);
          color: white;
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
