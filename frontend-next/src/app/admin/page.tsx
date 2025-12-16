'use client'

import { useState, useEffect } from 'react'
import Link from 'next/link'

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8084'

interface RateLimitStats {
  enabled: boolean
  requests_last_minute: number
  requests_last_hour: number
  requests_last_day: number
  limit_per_minute: number
  limit_per_hour: number
  limit_per_day: number
  remaining_this_minute: number
  remaining_this_hour: number
  remaining_this_day: number
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

export default function AdminPage() {
  const [rateLimitStats, setRateLimitStats] = useState<RateLimitStats | null>(null)
  const [recentChanges, setRecentChanges] = useState<PropertyChange[]>([])
  const [schedulerRunning, setSchedulerRunning] = useState(false)
  const [schedulerMessage, setSchedulerMessage] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchStats()
    fetchRecentChanges()

    // Auto-refresh stats every 10 seconds
    const interval = setInterval(() => {
      fetchStats()
    }, 10000)

    return () => clearInterval(interval)
  }, [])

  const fetchStats = async () => {
    try {
      const response = await fetch(`${API_URL}/api/ratelimit/stats`)
      if (response.ok) {
        const data = await response.json()
        setRateLimitStats(data)
      }
    } catch (error) {
      console.error('Error fetching stats:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchRecentChanges = async () => {
    try {
      const response = await fetch(`${API_URL}/api/changes/recent?limit=20`)
      if (response.ok) {
        const data = await response.json()
        setRecentChanges(data.changes || [])
      }
    } catch (error) {
      console.error('Error fetching changes:', error)
    }
  }

  const triggerScheduler = async () => {
    setSchedulerRunning(true)
    setSchedulerMessage('')

    try {
      const response = await fetch(`${API_URL}/api/scheduler/run`, {
        method: 'POST'
      })

      if (response.ok) {
        const data = await response.json()
        setSchedulerMessage(`✓ ${data.message}`)

        // Refresh stats and changes after a delay
        setTimeout(() => {
          fetchStats()
          fetchRecentChanges()
        }, 5000)
      } else {
        setSchedulerMessage('✗ スケジューラーの起動に失敗しました')
      }
    } catch (error) {
      setSchedulerMessage('✗ エラーが発生しました')
    } finally {
      setSchedulerRunning(false)
    }
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

  const getChangeTypeColor = (changeType: string) => {
    const colors: Record<string, string> = {
      'new_property': '#10b981',
      'rent_changed': '#f59e0b',
      'status_changed': '#6366f1',
      'property_removed': '#ef4444'
    }
    return colors[changeType] || '#6b7280'
  }

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)

    if (days > 0) return `${days}日前`
    if (hours > 0) return `${hours}時間前`
    if (minutes > 0) return `${minutes}分前`
    return '今'
  }

  const getProgressPercentage = (current: number, max: number) => {
    return Math.min((current / max) * 100, 100)
  }

  if (loading) {
    return (
      <div className="admin-page">
        <div style={{ textAlign: 'center', padding: '60px 20px' }}>
          <div className="loading"></div>
        </div>
      </div>
    )
  }

  return (
    <div className="admin-page">
      <header className="admin-header">
        <div className="header-inner">
          <Link href="/" className="back-button">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="m15 18-6-6 6-6"/>
            </svg>
            一覧に戻る
          </Link>
          <h1 className="admin-title">管理画面</h1>
        </div>
      </header>

      <main className="admin-content">
        <section className="admin-section">
          <h2 className="section-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M12 20h9"/>
              <path d="M16.5 3.5a2.12 2.12 0 0 1 3 3L7 19l-4 1 1-4Z"/>
            </svg>
            スケジューラー
          </h2>
          <div className="scheduler-card">
            <p className="scheduler-description">
              全アクティブ物件を再スクレイピングし、変更を検出します。
            </p>
            <button
              className="btn btn-primary btn-large"
              onClick={triggerScheduler}
              disabled={schedulerRunning}
            >
              {schedulerRunning ? (
                <>
                  <div className="spinner"></div>
                  実行中...
                </>
              ) : (
                <>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M5 12h14"/>
                    <path d="m12 5 7 7-7 7"/>
                  </svg>
                  スクレイピング実行
                </>
              )}
            </button>
            {schedulerMessage && (
              <div className={`scheduler-message ${schedulerMessage.startsWith('✓') ? 'success' : 'error'}`}>
                {schedulerMessage}
              </div>
            )}
          </div>
        </section>

        {rateLimitStats && (
          <section className="admin-section">
            <h2 className="section-title">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M3 3v18h18"/>
                <path d="m19 9-5 5-4-4-3 3"/>
              </svg>
              レート制限状況
            </h2>
            <div className="stats-grid">
              <div className="stat-card">
                <div className="stat-header">
                  <span className="stat-label">分間リクエスト</span>
                  <span className="stat-value">
                    {rateLimitStats.requests_last_minute} / {rateLimitStats.limit_per_minute}
                  </span>
                </div>
                <div className="progress-bar">
                  <div
                    className="progress-fill"
                    style={{
                      width: `${getProgressPercentage(rateLimitStats.requests_last_minute, rateLimitStats.limit_per_minute)}%`,
                      background: rateLimitStats.remaining_this_minute < 5 ? '#ef4444' : '#10b981'
                    }}
                  ></div>
                </div>
                <div className="stat-footer">
                  残り {rateLimitStats.remaining_this_minute} リクエスト
                </div>
              </div>

              <div className="stat-card">
                <div className="stat-header">
                  <span className="stat-label">時間リクエスト</span>
                  <span className="stat-value">
                    {rateLimitStats.requests_last_hour} / {rateLimitStats.limit_per_hour}
                  </span>
                </div>
                <div className="progress-bar">
                  <div
                    className="progress-fill"
                    style={{
                      width: `${getProgressPercentage(rateLimitStats.requests_last_hour, rateLimitStats.limit_per_hour)}%`,
                      background: rateLimitStats.remaining_this_hour < 100 ? '#f59e0b' : '#10b981'
                    }}
                  ></div>
                </div>
                <div className="stat-footer">
                  残り {rateLimitStats.remaining_this_hour} リクエスト
                </div>
              </div>

              <div className="stat-card">
                <div className="stat-header">
                  <span className="stat-label">日間リクエスト</span>
                  <span className="stat-value">
                    {rateLimitStats.requests_last_day} / {rateLimitStats.limit_per_day}
                  </span>
                </div>
                <div className="progress-bar">
                  <div
                    className="progress-fill"
                    style={{
                      width: `${getProgressPercentage(rateLimitStats.requests_last_day, rateLimitStats.limit_per_day)}%`,
                      background: '#10b981'
                    }}
                  ></div>
                </div>
                <div className="stat-footer">
                  残り {rateLimitStats.remaining_this_day} リクエスト
                </div>
              </div>
            </div>
          </section>
        )}

        <section className="admin-section">
          <h2 className="section-title">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <circle cx="12" cy="12" r="10"/>
              <polyline points="12 6 12 12 16 14"/>
            </svg>
            最近の変更 ({recentChanges.length})
          </h2>
          <div className="changes-list">
            {recentChanges.length === 0 ? (
              <div className="empty-state">
                <p>まだ変更はありません</p>
              </div>
            ) : (
              recentChanges.map((change) => (
                <div key={change.id} className="change-item">
                  <div
                    className="change-badge"
                    style={{ background: getChangeTypeColor(change.change_type) }}
                  >
                    {getChangeTypeLabel(change.change_type)}
                  </div>
                  <div className="change-content">
                    <div className="change-property-id">
                      <Link href={`/properties/${change.property_id}`}>
                        物件 ID: {change.property_id.slice(0, 8)}...
                      </Link>
                    </div>
                    {change.old_value && change.new_value && (
                      <div className="change-diff">
                        <span className="old-value">{change.old_value}</span>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M5 12h14"/>
                          <path d="m12 5 7 7-7 7"/>
                        </svg>
                        <span className="new-value">{change.new_value}</span>
                      </div>
                    )}
                  </div>
                  <div className="change-time">{formatDate(change.detected_at)}</div>
                </div>
              ))
            )}
          </div>
        </section>
      </main>

      <style jsx>{`
        .admin-page {
          min-height: 100vh;
          background: var(--bg-primary);
        }

        .admin-header {
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

        .admin-title {
          font-size: 20px;
          font-weight: 700;
          color: var(--text-primary);
          margin: 0;
        }

        .admin-content {
          max-width: 1200px;
          margin: 0 auto;
          padding: 40px 20px;
        }

        .admin-section {
          margin-bottom: 40px;
        }

        .section-title {
          font-size: 18px;
          font-weight: 700;
          color: var(--text-primary);
          margin: 0 0 20px 0;
          display: flex;
          align-items: center;
          gap: 12px;
        }

        .section-title svg {
          width: 20px;
          height: 20px;
        }

        .scheduler-card {
          background: white;
          border-radius: 12px;
          padding: 32px;
          text-align: center;
        }

        .scheduler-description {
          font-size: 14px;
          color: var(--text-muted);
          margin: 0 0 24px 0;
        }

        .btn-large {
          padding: 16px 32px;
          font-size: 16px;
          display: inline-flex;
          align-items: center;
          gap: 12px;
          min-width: 240px;
          justify-content: center;
        }

        .btn-large svg {
          width: 20px;
          height: 20px;
        }

        .spinner {
          width: 16px;
          height: 16px;
          border: 2px solid rgba(255, 255, 255, 0.3);
          border-top-color: white;
          border-radius: 50%;
          animation: spin 0.8s linear infinite;
        }

        @keyframes spin {
          to { transform: rotate(360deg); }
        }

        .scheduler-message {
          margin-top: 20px;
          padding: 12px 20px;
          border-radius: 8px;
          font-size: 14px;
          font-weight: 500;
        }

        .scheduler-message.success {
          background: #d1fae5;
          color: #065f46;
        }

        .scheduler-message.error {
          background: #fee2e2;
          color: #991b1b;
        }

        .stats-grid {
          display: grid;
          grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
          gap: 20px;
        }

        .stat-card {
          background: white;
          border-radius: 12px;
          padding: 24px;
        }

        .stat-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: 16px;
        }

        .stat-label {
          font-size: 14px;
          color: var(--text-muted);
          font-weight: 600;
        }

        .stat-value {
          font-size: 20px;
          color: var(--text-primary);
          font-weight: 700;
        }

        .progress-bar {
          height: 8px;
          background: var(--bg-secondary);
          border-radius: 4px;
          overflow: hidden;
          margin-bottom: 12px;
        }

        .progress-fill {
          height: 100%;
          border-radius: 4px;
          transition: width 0.3s ease;
        }

        .stat-footer {
          font-size: 13px;
          color: var(--text-muted);
        }

        .changes-list {
          background: white;
          border-radius: 12px;
          overflow: hidden;
        }

        .change-item {
          display: flex;
          align-items: center;
          gap: 16px;
          padding: 16px 20px;
          border-bottom: 1px solid var(--border-color);
        }

        .change-item:last-child {
          border-bottom: none;
        }

        .change-badge {
          padding: 6px 12px;
          border-radius: 6px;
          font-size: 12px;
          font-weight: 600;
          color: white;
          white-space: nowrap;
        }

        .change-content {
          flex: 1;
        }

        .change-property-id {
          font-size: 13px;
          margin-bottom: 4px;
        }

        .change-property-id a {
          color: var(--primary);
          text-decoration: none;
          font-weight: 600;
        }

        .change-property-id a:hover {
          text-decoration: underline;
        }

        .change-diff {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 13px;
          color: var(--text-muted);
        }

        .change-diff svg {
          width: 14px;
          height: 14px;
          flex-shrink: 0;
        }

        .old-value {
          text-decoration: line-through;
          opacity: 0.6;
        }

        .new-value {
          color: var(--text-primary);
          font-weight: 600;
        }

        .change-time {
          font-size: 12px;
          color: var(--text-muted);
          white-space: nowrap;
        }

        .empty-state {
          text-align: center;
          padding: 40px 20px;
          color: var(--text-muted);
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
      `}</style>
    </div>
  )
}
