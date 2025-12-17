import { useState, useEffect } from 'react'
import './App.css'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8084'

function App() {
  const [properties, setProperties] = useState([])
  const [searchQuery, setSearchQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [scrapeURL, setScrapeURL] = useState('')
  const [scraping, setScraping] = useState(false)

  // Filter states
  const [minRent, setMinRent] = useState('')
  const [maxRent, setMaxRent] = useState('')
  const [selectedFloorPlans, setSelectedFloorPlans] = useState([])
  const [maxWalkTime, setMaxWalkTime] = useState('')
  const [showFilters, setShowFilters] = useState(false)

  // Fetch properties on mount
  useEffect(() => {
    fetchProperties()
  }, [])

  const fetchProperties = async (query = '', filters = {}) => {
    setLoading(true)
    try {
      // Build query parameters
      const params = new URLSearchParams()

      if (query) params.append('q', query)
      if (filters.minRent) params.append('min_rent', filters.minRent)
      if (filters.maxRent) params.append('max_rent', filters.maxRent)
      if (filters.maxWalkTime) params.append('max_walk_time', filters.maxWalkTime)
      if (filters.floorPlans && filters.floorPlans.length > 0) {
        filters.floorPlans.forEach(plan => params.append('floor_plan', plan))
      }

      const hasFilters = params.toString().length > 0
      const endpoint = hasFilters ? '/api/filter' : '/api/properties'
      const url = `${API_URL}${endpoint}${hasFilters ? '?' + params.toString() : ''}`

      const response = await fetch(url)
      const data = await response.json()
      setProperties(data || [])
    } catch (error) {
      console.error('Error fetching properties:', error)
      alert('物件の取得に失敗しました')
    } finally {
      setLoading(false)
    }
  }

  const handleSearch = (e) => {
    e.preventDefault()
    const filters = {
      minRent,
      maxRent,
      maxWalkTime,
      floorPlans: selectedFloorPlans
    }
    fetchProperties(searchQuery, filters)
  }

  const handleResetFilters = () => {
    setSearchQuery('')
    setMinRent('')
    setMaxRent('')
    setSelectedFloorPlans([])
    setMaxWalkTime('')
    fetchProperties()
  }

  const toggleFloorPlan = (plan) => {
    setSelectedFloorPlans(prev =>
      prev.includes(plan)
        ? prev.filter(p => p !== plan)
        : [...prev, plan]
    )
  }

  const handleScrape = async (e) => {
    e.preventDefault()
    if (!scrapeURL.trim()) return

    setScraping(true)
    try {
      const response = await fetch(`${API_URL}/api/scrape`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ url: scrapeURL }),
      })

      if (!response.ok) {
        throw new Error('スクレイピングに失敗しました')
      }

      const data = await response.json()
      alert(`スクレイピング成功: ${data.title}`)
      setScrapeURL('')
      fetchProperties()
    } catch (error) {
      console.error('Error scraping:', error)
      alert('スクレイピングに失敗しました')
    } finally {
      setScraping(false)
    }
  }

  return (
    <div className="app">
      <header className="header">
        <h1>不動産ポータル - PoC</h1>
        <p className="subtitle">Yahoo不動産スクレイピング検証</p>
      </header>

      <main className="main">
        {/* Scrape Section */}
        <section className="scrape-section">
          <h2>物件URLをスクレイピング</h2>
          <form onSubmit={handleScrape} className="scrape-form">
            <input
              type="url"
              placeholder="Yahoo不動産の物件URLを入力"
              value={scrapeURL}
              onChange={(e) => setScrapeURL(e.target.value)}
              className="scrape-input"
              required
            />
            <button type="submit" disabled={scraping} className="scrape-button">
              {scraping ? 'スクレイピング中...' : 'スクレイピング'}
            </button>
          </form>
        </section>

        {/* Search & Filter Section */}
        <section className="search-section">
          <h2>物件検索・フィルタ</h2>
          <form onSubmit={handleSearch} className="search-form">
            <input
              type="text"
              placeholder="キーワードで検索..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="search-input"
            />
            <button
              type="button"
              onClick={() => setShowFilters(!showFilters)}
              className="filter-toggle-button"
            >
              {showFilters ? '▼ フィルタを隠す' : '▶ フィルタを表示'}
            </button>
            <button type="submit" disabled={loading} className="search-button">
              検索
            </button>
            <button
              type="button"
              onClick={handleResetFilters}
              className="reset-button"
            >
              リセット
            </button>
          </form>

          {/* Filter Panel */}
          {showFilters && (
            <div className="filter-panel">
              {/* Rent Filter */}
              <div className="filter-group">
                <label className="filter-label">賃料 (円)</label>
                <div className="filter-inputs">
                  <input
                    type="number"
                    placeholder="最小"
                    value={minRent}
                    onChange={(e) => setMinRent(e.target.value)}
                    className="filter-input"
                  />
                  <span className="filter-separator">〜</span>
                  <input
                    type="number"
                    placeholder="最大"
                    value={maxRent}
                    onChange={(e) => setMaxRent(e.target.value)}
                    className="filter-input"
                  />
                </div>
              </div>

              {/* Floor Plan Filter */}
              <div className="filter-group">
                <label className="filter-label">間取り</label>
                <div className="filter-checkboxes">
                  {['1K', '1DK', '1LDK', '2K', '2DK', '2LDK', '3LDK'].map(plan => (
                    <label key={plan} className="checkbox-label">
                      <input
                        type="checkbox"
                        checked={selectedFloorPlans.includes(plan)}
                        onChange={() => toggleFloorPlan(plan)}
                      />
                      <span>{plan}</span>
                    </label>
                  ))}
                </div>
              </div>

              {/* Walk Time Filter */}
              <div className="filter-group">
                <label className="filter-label">駅徒歩 (最大)</label>
                <select
                  value={maxWalkTime}
                  onChange={(e) => setMaxWalkTime(e.target.value)}
                  className="filter-select"
                >
                  <option value="">指定なし</option>
                  <option value="5">5分以内</option>
                  <option value="10">10分以内</option>
                  <option value="15">15分以内</option>
                  <option value="20">20分以内</option>
                </select>
              </div>
            </div>
          )}
        </section>

        {/* Results Section */}
        <section className="results-section">
          <h2>物件一覧 ({properties.length}件)</h2>
          {loading ? (
            <div className="loading">読み込み中...</div>
          ) : properties.length === 0 ? (
            <div className="no-results">物件が見つかりませんでした</div>
          ) : (
            <div className="property-grid">
              {properties.map((property) => (
                <PropertyCard key={property.id} property={property} />
              ))}
            </div>
          )}
        </section>
      </main>
    </div>
  )
}

function PropertyCard({ property }) {
  const [imageError, setImageError] = useState(false)

  const formatRent = (rent) => {
    if (!rent) return null
    return (rent / 10000).toFixed(1) + '万円'
  }

  return (
    <div className="property-card">
      <div className="property-image-container">
        {property.image_url && !imageError ? (
          <img
            src={property.image_url}
            alt={property.title}
            className="property-image"
            onError={() => setImageError(true)}
          />
        ) : (
          <div className="property-image-placeholder">
            画像なし
          </div>
        )}
      </div>
      <div className="property-info">
        <h3 className="property-title">{property.title}</h3>

        {/* Property Details */}
        <div className="property-details">
          {property.rent && (
            <div className="property-detail">
              <span className="detail-label">賃料:</span>
              <span className="detail-value">{formatRent(property.rent)}</span>
            </div>
          )}
          {property.floor_plan && (
            <div className="property-detail">
              <span className="detail-label">間取り:</span>
              <span className="detail-value">{property.floor_plan}</span>
            </div>
          )}
          {property.area && (
            <div className="property-detail">
              <span className="detail-label">面積:</span>
              <span className="detail-value">{property.area}㎡</span>
            </div>
          )}
          {property.walk_time && (
            <div className="property-detail">
              <span className="detail-label">駅徒歩:</span>
              <span className="detail-value">{property.walk_time}分</span>
            </div>
          )}
          {property.building_age && (
            <div className="property-detail">
              <span className="detail-label">築年数:</span>
              <span className="detail-value">築{property.building_age}年</span>
            </div>
          )}
        </div>

        <a
          href={property.detail_url}
          target="_blank"
          rel="noreferrer"
          className="property-link"
        >
          Yahoo不動産で詳細を見る →
        </a>
      </div>
    </div>
  )
}

export default App
