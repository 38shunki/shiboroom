'use client'

import { useState, useEffect } from 'react'
import { useRouter } from 'next/navigation'

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

type ViewMode = 'search' | 'results' | 'compare'
type PropertyStatus = 'none' | 'candidate' | 'maybe' | 'excluded'
type SortOption = 'newest' | 'rent_asc' | 'rent_desc' | 'area_desc' | 'walk_time_asc' | 'building_age_asc'
type DisplayMode = 'grid' | 'list' | 'map'

export default function Home() {
  const [currentView, setCurrentView] = useState<ViewMode>('results')
  const [properties, setProperties] = useState<Property[]>([])
  const [loading, setLoading] = useState(false)
  const [propertyStatuses, setPropertyStatuses] = useState<Record<string, PropertyStatus>>({})

  // Search & Filter states
  const [searchQuery, setSearchQuery] = useState('')
  const [minRent, setMinRent] = useState<number>(5)
  const [maxRent, setMaxRent] = useState<number>(50)
  const [selectedFloorPlans, setSelectedFloorPlans] = useState<string[]>([])
  const [minWalkTime, setMinWalkTime] = useState<number>(1)
  const [maxWalkTime, setMaxWalkTime] = useState<number>(30)
  const [minArea, setMinArea] = useState<number>(15)
  const [maxArea, setMaxArea] = useState<number>(100)
  const [minBuildingAge, setMinBuildingAge] = useState<number>(0)
  const [maxBuildingAge, setMaxBuildingAge] = useState<number>(50)
  const [minFloor, setMinFloor] = useState<number>(1)
  const [maxFloor, setMaxFloor] = useState<number>(30)

  // Additional filters
  const [selectedBuildingTypes, setSelectedBuildingTypes] = useState<string[]>([])
  const [selectedFeatures, setSelectedFeatures] = useState<string[]>([])
  const [selectedStatuses, setSelectedStatuses] = useState<string[]>(['none', 'candidate', 'maybe'])

  // View options
  const [sortBy, setSortBy] = useState<SortOption>('newest')
  const [displayMode, setDisplayMode] = useState<DisplayMode>('grid')

  useEffect(() => {
    fetchProperties()
  }, [])

  // Re-fetch when sort changes
  useEffect(() => {
    if (properties.length > 0) {
      const filters = {
        minRent,
        maxRent,
        minWalkTime,
        maxWalkTime,
        floorPlans: selectedFloorPlans
      }
      fetchProperties(searchQuery, filters)
    }
  }, [sortBy])

  const fetchProperties = async (query = '', filters: any = {}) => {
    setLoading(true)
    try {
      // Use advanced search endpoint with Meilisearch
      const searchPayload = {
        query: query || '',
        min_rent: filters.minRent ? filters.minRent * 10000 : undefined,
        max_rent: filters.maxRent ? filters.maxRent * 10000 : undefined,
        min_walk_time: filters.minWalkTime,
        max_walk_time: filters.maxWalkTime,
        floor_plans: filters.floorPlans && filters.floorPlans.length > 0 ? filters.floorPlans : undefined,
        sort: sortBy,
        limit: 100,
        facets: ['floor_plan', 'station']
      }

      // Remove undefined values
      Object.keys(searchPayload).forEach(key =>
        searchPayload[key as keyof typeof searchPayload] === undefined && delete searchPayload[key as keyof typeof searchPayload]
      )

      const response = await fetch(`${API_URL}/api/search/advanced`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(searchPayload),
      })

      const data = await response.json()
      setProperties(data.hits || [])
    } catch (error) {
      console.error('Error fetching properties:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    const filters = {
      minRent,
      maxRent,
      minWalkTime,
      maxWalkTime,
      floorPlans: selectedFloorPlans
    }
    fetchProperties(searchQuery, filters)
    setCurrentView('results')
  }

  const handleResetFilters = () => {
    setSearchQuery('')
    setMinRent(3)
    setMaxRent(30)
    setSelectedFloorPlans([])
    setMinWalkTime(1)
    setMaxWalkTime(30)
    setMinArea(15)
    setMaxArea(100)
    setMinBuildingAge(0)
    setMaxBuildingAge(30)
    setMinFloor(1)
    setMaxFloor(20)
    setSelectedBuildingTypes(['マンション', 'アパート'])
    setSelectedFeatures([])
    setSelectedStatuses(['none', 'candidate', 'maybe'])
    fetchProperties()
  }

  const toggleFloorPlan = (plan: string) => {
    setSelectedFloorPlans(prev =>
      prev.includes(plan)
        ? prev.filter(p => p !== plan)
        : [...prev, plan]
    )
  }

  const toggleBuildingType = (type: string) => {
    setSelectedBuildingTypes(prev =>
      prev.includes(type)
        ? prev.filter(t => t !== type)
        : [...prev, type]
    )
  }

  const toggleFeature = (feature: string) => {
    setSelectedFeatures(prev =>
      prev.includes(feature)
        ? prev.filter(f => f !== feature)
        : [...prev, feature]
    )
  }

  const toggleStatusFilter = (status: string) => {
    setSelectedStatuses(prev =>
      prev.includes(status)
        ? prev.filter(s => s !== status)
        : [...prev, status]
    )
  }

  const setPropertyStatus = (propertyId: string, status: PropertyStatus) => {
    setPropertyStatuses(prev => ({
      ...prev,
      [propertyId]: prev[propertyId] === status ? 'none' : status
    }))
  }

  const formatRent = (rent?: number) => {
    if (!rent) return null
    return (rent / 10000).toFixed(1)
  }

  const candidateCount = Object.values(propertyStatuses).filter(s => s === 'candidate').length
  const candidateProperties = properties.filter(p => propertyStatuses[p.id] === 'candidate')

  // Filter properties (client-side filters for UI interactions)
  const filteredAndSortedProperties = () => {
    let filtered = [...properties]

    // Apply sidebar filters with range (client-side refinement)
    filtered = filtered.filter(p => {
      // Rent filter
      if (p.rent) {
        const rentInMan = p.rent / 10000
        if (rentInMan < minRent || rentInMan > maxRent) return false
      }

      // Area filter
      if (p.area) {
        if (p.area < minArea || p.area > maxArea) return false
      }

      // Building age filter
      if (p.building_age !== undefined) {
        if (p.building_age < minBuildingAge || p.building_age > maxBuildingAge) return false
      }

      // Walk time filter
      if (p.walk_time) {
        if (p.walk_time < minWalkTime || p.walk_time > maxWalkTime) return false
      }

      // Floor filter
      if (p.floor) {
        if (p.floor < minFloor || p.floor > maxFloor) return false
      }

      return true
    })

    // Floor plan filter
    if (selectedFloorPlans.length > 0) {
      filtered = filtered.filter(p => p.floor_plan && selectedFloorPlans.includes(p.floor_plan))
    }

    // Status filter
    if (selectedStatuses.length > 0) {
      filtered = filtered.filter(p => {
        const status = propertyStatuses[p.id] || 'none'
        return selectedStatuses.includes(status)
      })
    }

    // Sorting is handled by Meilisearch on the server
    return filtered
  }

  const displayedProperties = filteredAndSortedProperties()

  return (
    <div className="app">
      <header className="header">
        <div className="header-inner">
          <a href="#" className="logo" onClick={() => setCurrentView('search')}>shiboroom<span>.</span></a>
          <nav className="nav">
            <button
              className={`nav-btn ${currentView === 'search' ? 'active' : ''}`}
              onClick={() => setCurrentView('search')}
            >
              検索
            </button>
            <button
              className={`nav-btn ${currentView === 'results' ? 'active' : ''}`}
              onClick={() => setCurrentView('results')}
            >
              一覧
            </button>
            <button
              className={`nav-btn ${currentView === 'compare' ? 'active' : ''}`}
              onClick={() => setCurrentView('compare')}
            >
              比較
            </button>
          </nav>
          <div className="header-actions">
            <div className="count-badge">
              <span className="item">全 <strong className="accent">{properties.length}</strong> 件</span>
              <span className="item">候補 <strong className="candidate-count">{candidateCount}</strong> 件</span>
            </div>
            <a href="/admin" className="admin-link">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/>
                <circle cx="12" cy="12" r="3"/>
              </svg>
            </a>
          </div>
        </div>
      </header>

      {/* Search View */}
      {currentView === 'search' && (
        <section className="view active" id="search">
          <div className="landing">
            <h1 className="landing-title">物件を<em>ふるい分け</em>する</h1>
            <p className="landing-desc">条件を入力して検索。ヒットした物件を比較して、本当に検討すべき物件だけを残せます。</p>

            <div className="search-card">
              <form onSubmit={handleSearch}>
                <div className="search-section">
                  <h3 className="search-section-title">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                    キーワード検索
                  </h3>
                  <div className="search-row">
                    <div className="form-group" style={{ width: '100%' }}>
                      <label className="form-label">駅名・エリア・住所</label>
                      <input
                        type="text"
                        placeholder="例: 新宿、渋谷、市ヶ谷"
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="form-input"
                        style={{ width: '100%' }}
                      />
                    </div>
                  </div>
                </div>

                <div className="search-section">
                  <h3 className="search-section-title">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                    賃料・広さ
                  </h3>
                  <div className="search-row">
                    <div className="form-group">
                      <label className="form-label">賃料（万円）</label>
                      <div className="range-group">
                        <input
                          type="number"
                          placeholder="下限なし"
                          value={minRent}
                          onChange={(e) => setMinRent(Number(e.target.value))}
                          className="form-input"
                        />
                        <span className="range-separator">〜</span>
                        <input
                          type="number"
                          placeholder="上限なし"
                          value={maxRent}
                          onChange={(e) => setMaxRent(Number(e.target.value))}
                          className="form-input"
                        />
                      </div>
                    </div>
                  </div>
                </div>

                <div className="search-section">
                  <h3 className="search-section-title">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
                    間取り・建物
                  </h3>
                  <div className="form-group">
                    <label className="form-label">間取り</label>
                    <div className="tag-group">
                      {['ワンルーム', '1K', '1DK', '1LDK', '2K', '2DK', '2LDK', '3K', '3DK', '3LDK'].map(plan => (
                        <label key={plan} className="tag-item">
                          <input
                            type="checkbox"
                            checked={selectedFloorPlans.includes(plan)}
                            onChange={() => toggleFloorPlan(plan)}
                          />
                          <span className="tag-label">{plan}</span>
                        </label>
                      ))}
                    </div>
                  </div>
                  <div className="search-row" style={{ marginTop: '16px' }}>
                    <div className="form-group">
                      <label className="form-label">駅徒歩</label>
                      <select value={maxWalkTime} onChange={(e) => setMaxWalkTime(Number(e.target.value))} className="form-select">
                        <option value="30">指定なし</option>
                        <option value="5">5分以内</option>
                        <option value="10">10分以内</option>
                        <option value="15">15分以内</option>
                        <option value="20">20分以内</option>
                      </select>
                    </div>
                  </div>
                </div>

                <div className="search-actions">
                  <span className="search-hint">検索結果から更に絞り込みできます</span>
                  <div className="search-buttons">
                    <button type="button" className="btn btn-secondary" onClick={handleResetFilters}>条件クリア</button>
                    <button type="submit" className="btn btn-primary" disabled={loading}>
                      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
                      この条件で検索
                    </button>
                  </div>
                </div>
              </form>
            </div>

            <div className="features">
              <div className="feature">
                <div className="feature-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/></svg>
                </div>
                <h3>条件で絞る</h3>
                <p>結果一覧から、さらにスライダーやチェックで即座に絞り込み</p>
              </div>
              <div className="feature">
                <div className="feature-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M20 6 9 17l-5-5"/></svg>
                </div>
                <h3>判断を整理</h3>
                <p>候補・微妙・除外の3段階で、検討状況を一目で把握</p>
              </div>
              <div className="feature">
                <div className="feature-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>
                </div>
                <h3>詳細は元サイトで</h3>
                <p>問い合わせ・詳細確認はYahoo不動産の物件ページへ</p>
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Results View */}
      {currentView === 'results' && (
        <section className="results-view active">
          {/* Sidebar Filters */}
          <aside className="sidebar">
            <div className="sidebar-header">
              <span className="sidebar-title">条件で絞り込み</span>
              <button className="edit-search-btn" onClick={() => setCurrentView('search')}>検索条件を編集</button>
            </div>
            <div className="sidebar-content">
              <div className="filter-section">
                <h3 className="filter-title">リアルタイム絞り込み</h3>

                <div className="filter-group">
                  <label className="filter-label">
                    家賃
                    <span className="filter-value">{minRent}〜{maxRent}万円</span>
                  </label>
                  <div className="dual-range">
                    <input
                      type="range"
                      className="range-slider"
                      min="3"
                      max="50"
                      value={minRent}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val < maxRent) setMinRent(val)
                      }}
                    />
                    <input
                      type="range"
                      className="range-slider"
                      min="3"
                      max="50"
                      value={maxRent}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val > minRent) setMaxRent(val)
                      }}
                    />
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">
                    駅徒歩
                    <span className="filter-value">{minWalkTime}〜{maxWalkTime}分</span>
                  </label>
                  <div className="dual-range">
                    <input
                      type="range"
                      className="range-slider"
                      min="1"
                      max="30"
                      value={minWalkTime}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val < maxWalkTime) setMinWalkTime(val)
                      }}
                    />
                    <input
                      type="range"
                      className="range-slider"
                      min="1"
                      max="30"
                      value={maxWalkTime}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val > minWalkTime) setMaxWalkTime(val)
                      }}
                    />
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">
                    専有面積
                    <span className="filter-value">{minArea}〜{maxArea}㎡</span>
                  </label>
                  <div className="dual-range">
                    <input
                      type="range"
                      className="range-slider"
                      min="15"
                      max="100"
                      value={minArea}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val < maxArea) setMinArea(val)
                      }}
                    />
                    <input
                      type="range"
                      className="range-slider"
                      min="15"
                      max="100"
                      value={maxArea}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val > minArea) setMaxArea(val)
                      }}
                    />
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">
                    築年数
                    <span className="filter-value">{minBuildingAge}〜{maxBuildingAge}年</span>
                  </label>
                  <div className="dual-range">
                    <input
                      type="range"
                      className="range-slider"
                      min="0"
                      max="50"
                      value={minBuildingAge}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val < maxBuildingAge) setMinBuildingAge(val)
                      }}
                    />
                    <input
                      type="range"
                      className="range-slider"
                      min="0"
                      max="50"
                      value={maxBuildingAge}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val > minBuildingAge) setMaxBuildingAge(val)
                      }}
                    />
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">
                    階数
                    <span className="filter-value">{minFloor}〜{maxFloor}階</span>
                  </label>
                  <div className="dual-range">
                    <input
                      type="range"
                      className="range-slider"
                      min="1"
                      max="30"
                      value={minFloor}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val < maxFloor) setMinFloor(val)
                      }}
                    />
                    <input
                      type="range"
                      className="range-slider"
                      min="1"
                      max="30"
                      value={maxFloor}
                      onChange={(e) => {
                        const val = Number(e.target.value)
                        if (val > minFloor) setMaxFloor(val)
                      }}
                    />
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">間取り</label>
                  <div className="checkbox-group">
                    {['1R', '1K', '1DK', '1LDK', '2K', '2DK', '2LDK', '3K', '3DK', '3LDK'].map(plan => (
                      <label key={plan} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFloorPlans.includes(plan)}
                          onChange={() => toggleFloorPlan(plan)}
                        />
                        <span className="checkbox-label">{plan}</span>
                      </label>
                    ))}
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">建物種別</label>
                  <div className="checkbox-group">
                    {['マンション', 'アパート', '一戸建て'].map(type => (
                      <label key={type} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedBuildingTypes.includes(type)}
                          onChange={() => toggleBuildingType(type)}
                        />
                        <span className="checkbox-label">{type}</span>
                      </label>
                    ))}
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">こだわり条件</label>
                  <div className="checkbox-group">
                    {['バストイレ別', 'オートロック', '宅配BOX', 'ペット可', '南向き', '角部屋', '2階以上', '駐車場あり'].map(feature => (
                      <label key={feature} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedFeatures.includes(feature)}
                          onChange={() => toggleFeature(feature)}
                        />
                        <span className="checkbox-label">{feature}</span>
                      </label>
                    ))}
                  </div>
                </div>

                <div className="filter-group">
                  <label className="filter-label">ステータス</label>
                  <div className="checkbox-group">
                    {[
                      { key: 'none', label: '未分類' },
                      { key: 'candidate', label: '候補' },
                      { key: 'maybe', label: '微妙' },
                      { key: 'excluded', label: '除外' }
                    ].map(({ key, label }) => (
                      <label key={key} className="checkbox-item">
                        <input
                          type="checkbox"
                          checked={selectedStatuses.includes(key)}
                          onChange={() => toggleStatusFilter(key)}
                        />
                        <span className="checkbox-label">{label}</span>
                      </label>
                    ))}
                  </div>
                </div>
              </div>
              <div className="filter-result">
                <div className="filter-result-count">{displayedProperties.length}</div>
                <div className="filter-result-label">件がヒット</div>
              </div>
              <button className="reset-btn" onClick={handleResetFilters}>絞り込みをリセット</button>
            </div>
          </aside>

          <main className="main-content">
            <div className="toolbar">
              <div className="toolbar-left">
                <div className="search-status">
                  {loading ? (
                    <><span className="loading"></span> 読み込み中...</>
                  ) : (
                    `${displayedProperties.length}件の物件`
                  )}
                </div>
              </div>
              <div className="toolbar-left">
                <select value={sortBy} onChange={(e) => setSortBy(e.target.value as SortOption)} className="sort-select">
                  <option value="newest">新着順</option>
                  <option value="rent_asc">家賃が安い順</option>
                  <option value="walk_time_asc">駅から近い順</option>
                  <option value="building_age_asc">築年数が新しい順</option>
                  <option value="area_desc">専有面積が広い順</option>
                </select>
                <div className="view-toggle">
                  <button
                    className={`view-toggle-btn ${displayMode === 'grid' ? 'active' : ''}`}
                    onClick={() => setDisplayMode('grid')}
                    title="グリッド表示"
                  >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <rect width="7" height="7" x="3" y="3" rx="1"/>
                      <rect width="7" height="7" x="14" y="3" rx="1"/>
                      <rect width="7" height="7" x="14" y="14" rx="1"/>
                      <rect width="7" height="7" x="3" y="14" rx="1"/>
                    </svg>
                  </button>
                  <button
                    className={`view-toggle-btn ${displayMode === 'list' ? 'active' : ''}`}
                    onClick={() => setDisplayMode('list')}
                    title="リスト表示"
                  >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <line x1="8" x2="21" y1="6" y2="6"/>
                      <line x1="8" x2="21" y1="12" y2="12"/>
                      <line x1="8" x2="21" y1="18" y2="18"/>
                      <line x1="3" x2="3.01" y1="6" y2="6"/>
                      <line x1="3" x2="3.01" y1="12" y2="12"/>
                      <line x1="3" x2="3.01" y1="18" y2="18"/>
                    </svg>
                  </button>
                  <button
                    className={`view-toggle-btn ${displayMode === 'map' ? 'active' : ''}`}
                    onClick={() => setDisplayMode('map')}
                    title="地図表示"
                  >
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                      <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/>
                      <circle cx="12" cy="10" r="3"/>
                    </svg>
                  </button>
                </div>
              </div>
            </div>

            {displayMode === 'map' && (
              <div className="map-container">
                <div className="map-placeholder">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/>
                    <circle cx="12" cy="10" r="3"/>
                  </svg>
                  <p>地図表示機能は開発中です</p>
                </div>
              </div>
            )}

            <div className="property-grid">
              {displayedProperties.map((property) => (
                <PropertyCard
                  key={property.id}
                  property={property}
                  formatRent={formatRent}
                  status={propertyStatuses[property.id] || 'none'}
                  onStatusChange={(status) => setPropertyStatus(property.id, status)}
                />
              ))}
            </div>
          </main>
        </section>
      )}

      {/* Compare View */}
      {currentView === 'compare' && (
        <section className="compare-view active">
          <div className="compare-header">
            <h2 className="compare-title">候補物件 <span>{candidateCount}件</span></h2>
          </div>
          <div className="compare-table">
            <table>
              <thead>
                <tr>
                  <th>物件</th>
                  <th>賃料</th>
                  <th>間取り</th>
                  <th>最寄駅</th>
                  <th>築年数</th>
                  <th></th>
                </tr>
              </thead>
              <tbody>
                {candidateProperties.map(property => (
                  <tr key={property.id}>
                    <td>
                      <div className="compare-property">
                        {property.image_url && (
                          <img src={property.image_url} className="compare-property-img" alt="" />
                        )}
                        <div>
                          <div className="compare-property-name">{property.title}</div>
                          <div className="compare-property-address">{property.address}</div>
                        </div>
                      </div>
                    </td>
                    <td>
                      {property.rent && (
                        <span className="compare-value highlight">{formatRent(property.rent)}万円</span>
                      )}
                    </td>
                    <td>
                      <span className="compare-value">
                        {property.floor_plan} / {property.area}㎡
                      </span>
                    </td>
                    <td>
                      <span className="compare-value">
                        {property.station} 徒歩{property.walk_time}分
                      </span>
                    </td>
                    <td>
                      <span className="compare-value">築{property.building_age}年</span>
                    </td>
                    <td>
                      <a href={property.detail_url} className="btn btn-secondary" target="_blank" rel="noreferrer">
                        詳細
                      </a>
                    </td>
                  </tr>
                ))}
                {candidateProperties.length === 0 && (
                  <tr>
                    <td colSpan={6} style={{ textAlign: 'center', padding: '40px', color: 'var(--text-muted)' }}>
                      候補に追加された物件はありません
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  )
}

function PropertyCard({
  property,
  formatRent,
  status,
  onStatusChange
}: {
  property: Property
  formatRent: (rent?: number) => string | null
  status: PropertyStatus
  onStatusChange: (status: PropertyStatus) => void
}) {
  const [imageError, setImageError] = useState(false)
  const router = useRouter()

  const handleCardClick = (e: React.MouseEvent) => {
    // Don't navigate if clicking on buttons
    if ((e.target as HTMLElement).closest('button, a')) {
      return
    }
    router.push(`/properties/${property.id}`)
  }

  return (
    <article
      className={`property-card ${status !== 'none' ? `status-${status}` : ''}`}
      onClick={handleCardClick}
      style={{ cursor: 'pointer' }}
    >
      <div className="property-image">
        {property.image_url && !imageError ? (
          <img
            src={property.image_url}
            alt={property.title}
            onError={() => setImageError(true)}
          />
        ) : (
          <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-tertiary)', color: 'var(--text-muted)', fontSize: '13px' }}>
            画像なし
          </div>
        )}
      </div>
      <div className="property-body">
        {property.rent && (
          <div className="property-price">
            {formatRent(property.rent)}<small>万円</small>
          </div>
        )}
        <div className="property-meta">
          {property.floor_plan && (
            <span>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/></svg>
              {property.floor_plan}
            </span>
          )}
          {property.area && <span>{property.area}㎡</span>}
          {property.building_age && <span>築{property.building_age}年</span>}
          {property.floor && <span>{property.floor}階</span>}
        </div>
        {(property.station || property.address) && (
          <div className="property-location">
            {property.station && property.walk_time && (
              <><strong>{property.station}</strong> 徒歩{property.walk_time}分</>
            )}
            {property.address && <> ／ {property.address}</>}
          </div>
        )}
        <div className="property-actions">
          <button
            className={`status-btn candidate ${status === 'candidate' ? 'active' : ''}`}
            onClick={() => onStatusChange('candidate')}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M20 6 9 17l-5-5"/></svg>
            候補
          </button>
          <button
            className={`status-btn maybe ${status === 'maybe' ? 'active' : ''}`}
            onClick={() => onStatusChange('maybe')}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><circle cx="12" cy="12" r="10"/><line x1="12" x2="12" y1="8" y2="12"/><line x1="12" x2="12.01" y1="16" y2="16"/></svg>
            微妙
          </button>
          <button
            className={`status-btn exclude ${status === 'excluded' ? 'active' : ''}`}
            onClick={() => onStatusChange('excluded')}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
            除外
          </button>
        </div>
        <div className="property-footer">
          <a href={property.detail_url} className="external-link" target="_blank" rel="noreferrer">
            Yahoo不動産で詳細を見る
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>
          </a>
        </div>
      </div>
    </article>
  )
}
