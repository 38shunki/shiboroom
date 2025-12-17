'use client'

import { useState, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

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
  const router = useRouter()
  const searchParams = useSearchParams()

  // Helper function to get array from URL params
  const getArrayParam = (key: string, defaultValue: string[] = []): string[] => {
    const value = searchParams.get(key)
    return value ? value.split(',').filter(Boolean) : defaultValue
  }

  // Helper function to get number from URL params
  const getNumberParam = (key: string, defaultValue: number): number => {
    const value = searchParams.get(key)
    return value ? Number(value) : defaultValue
  }

  // Initialize state from URL params or defaults
  const [currentView, setCurrentView] = useState<ViewMode>(
    (searchParams.get('view') as ViewMode) || 'search'
  )
  const [properties, setProperties] = useState<Property[]>([])
  const [loading, setLoading] = useState(false)
  const [propertyStatuses, setPropertyStatuses] = useState<Record<string, PropertyStatus>>({})
  const [hiddenProperties, setHiddenProperties] = useState<Set<string>>(new Set())
  const [propertyMemos, setPropertyMemos] = useState<Record<string, string>>({})

  // Load property statuses from localStorage on mount
  useEffect(() => {
    const savedStatuses = localStorage.getItem('propertyStatuses')
    if (savedStatuses) {
      try {
        setPropertyStatuses(JSON.parse(savedStatuses))
      } catch (error) {
        console.error('Failed to load property statuses:', error)
      }
    }

    const savedHidden = localStorage.getItem('hiddenProperties')
    if (savedHidden) {
      try {
        setHiddenProperties(new Set(JSON.parse(savedHidden)))
      } catch (error) {
        console.error('Failed to load hidden properties:', error)
      }
    }

    const savedMemos = localStorage.getItem('propertyMemos')
    if (savedMemos) {
      try {
        setPropertyMemos(JSON.parse(savedMemos))
      } catch (error) {
        console.error('Failed to load property memos:', error)
      }
    }
  }, [])

  // Save property statuses to localStorage whenever they change
  useEffect(() => {
    if (Object.keys(propertyStatuses).length > 0) {
      localStorage.setItem('propertyStatuses', JSON.stringify(propertyStatuses))
    }
  }, [propertyStatuses])

  // Save hidden properties to localStorage whenever they change
  useEffect(() => {
    if (hiddenProperties.size > 0) {
      localStorage.setItem('hiddenProperties', JSON.stringify(Array.from(hiddenProperties)))
    }
  }, [hiddenProperties])

  // Save property memos to localStorage whenever they change
  useEffect(() => {
    if (Object.keys(propertyMemos).length > 0) {
      localStorage.setItem('propertyMemos', JSON.stringify(propertyMemos))
    }
  }, [propertyMemos])

  // Search & Filter states (initialized from URL params)
  const [searchQuery, setSearchQuery] = useState(searchParams.get('q') || '')
  const [minRent, setMinRent] = useState<number>(getNumberParam('minRent', 3))
  const [maxRent, setMaxRent] = useState<number>(getNumberParam('maxRent', 50))
  const [selectedFloorPlans, setSelectedFloorPlans] = useState<string[]>(getArrayParam('floorPlans'))
  const [minWalkTime, setMinWalkTime] = useState<number>(getNumberParam('minWalkTime', 1))
  const [maxWalkTime, setMaxWalkTime] = useState<number>(getNumberParam('maxWalkTime', 30))
  const [minArea, setMinArea] = useState<number>(getNumberParam('minArea', 15))
  const [maxArea, setMaxArea] = useState<number>(getNumberParam('maxArea', 100))
  const [minBuildingAge, setMinBuildingAge] = useState<number>(getNumberParam('minBuildingAge', 0))
  const [maxBuildingAge, setMaxBuildingAge] = useState<number>(getNumberParam('maxBuildingAge', 50))
  const [minFloor, setMinFloor] = useState<number>(getNumberParam('minFloor', 1))
  const [maxFloor, setMaxFloor] = useState<number>(getNumberParam('maxFloor', 30))

  // Additional filters
  const [selectedBuildingTypes, setSelectedBuildingTypes] = useState<string[]>(getArrayParam('buildingTypes'))
  const [selectedFeatures, setSelectedFeatures] = useState<string[]>(getArrayParam('features'))
  const [selectedStatuses, setSelectedStatuses] = useState<string[]>(
    getArrayParam('statuses', ['none', 'candidate', 'maybe'])
  )

  // View options
  const [sortBy, setSortBy] = useState<SortOption>(
    (searchParams.get('sort') as SortOption) || 'newest'
  )
  const [displayMode, setDisplayMode] = useState<DisplayMode>(
    (searchParams.get('display') as DisplayMode) || 'grid'
  )

  // Update URL when filters change
  useEffect(() => {
    const params = new URLSearchParams()

    // Add view
    if (currentView !== 'search') {
      params.set('view', currentView)
    }

    // Add search query
    if (searchQuery) params.set('q', searchQuery)

    // Add filters (only if different from defaults)
    if (minRent !== 3) params.set('minRent', minRent.toString())
    if (maxRent !== 50) params.set('maxRent', maxRent.toString())
    if (selectedFloorPlans.length > 0) params.set('floorPlans', selectedFloorPlans.join(','))
    if (minWalkTime !== 1) params.set('minWalkTime', minWalkTime.toString())
    if (maxWalkTime !== 30) params.set('maxWalkTime', maxWalkTime.toString())
    if (minArea !== 15) params.set('minArea', minArea.toString())
    if (maxArea !== 100) params.set('maxArea', maxArea.toString())
    if (minBuildingAge !== 0) params.set('minBuildingAge', minBuildingAge.toString())
    if (maxBuildingAge !== 50) params.set('maxBuildingAge', maxBuildingAge.toString())
    if (minFloor !== 1) params.set('minFloor', minFloor.toString())
    if (maxFloor !== 30) params.set('maxFloor', maxFloor.toString())
    if (selectedBuildingTypes.length > 0) params.set('buildingTypes', selectedBuildingTypes.join(','))
    if (selectedFeatures.length > 0) params.set('features', selectedFeatures.join(','))

    // Status filter (only if different from default)
    const defaultStatuses = ['none', 'candidate', 'maybe'].sort().join(',')
    const currentStatuses = [...selectedStatuses].sort().join(',')
    if (currentStatuses !== defaultStatuses) {
      params.set('statuses', selectedStatuses.join(','))
    }

    // View options
    if (sortBy !== 'newest') params.set('sort', sortBy)
    if (displayMode !== 'grid') params.set('display', displayMode)

    // Update URL without adding to history (using replace)
    const queryString = params.toString()
    const newUrl = queryString ? `/?${queryString}` : '/'
    router.replace(newUrl, { scroll: false })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    currentView,
    searchQuery,
    minRent,
    maxRent,
    selectedFloorPlans,
    minWalkTime,
    maxWalkTime,
    minArea,
    maxArea,
    minBuildingAge,
    maxBuildingAge,
    minFloor,
    maxFloor,
    selectedBuildingTypes,
    selectedFeatures,
    selectedStatuses,
    sortBy,
    displayMode
    // Intentionally not including router to avoid infinite loop
  ])

  // Initial fetch on mount
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
      // Use simple properties endpoint (Meilisearch not available yet)
      const response = await fetch(`${API_URL}/api/properties?limit=1000`)

      const data = await response.json()

      // API returns array directly, not wrapped in {properties: [...]}
      const propertiesArray = Array.isArray(data) ? data : (data.properties || [])
      console.log('✅ Fetched properties:', propertiesArray.length)
      setProperties(propertiesArray)
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
    setMaxRent(50)
    setSelectedFloorPlans([])
    setMinWalkTime(1)
    setMaxWalkTime(30)
    setMinArea(15)
    setMaxArea(100)
    setMinBuildingAge(0)
    setMaxBuildingAge(50)
    setMinFloor(1)
    setMaxFloor(30)
    setSelectedBuildingTypes([])
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

  const hideProperty = (propertyId: string) => {
    setHiddenProperties(prev => {
      const newSet = new Set(prev)
      newSet.add(propertyId)
      return newSet
    })
  }

  const unhideProperty = (propertyId: string) => {
    setHiddenProperties(prev => {
      const newSet = new Set(prev)
      newSet.delete(propertyId)
      return newSet
    })
  }

  const setPropertyMemo = (propertyId: string, memo: string) => {
    setPropertyMemos(prev => {
      if (memo.trim() === '') {
        // Remove memo if empty
        const newMemos = { ...prev }
        delete newMemos[propertyId]
        return newMemos
      }
      return {
        ...prev,
        [propertyId]: memo
      }
    })
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

    // Filter out hidden properties
    filtered = filtered.filter(p => !hiddenProperties.has(p.id))

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
              ホーム
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

      {/* Home View */}
      {currentView === 'search' && (
        <section className="lp-view active" id="home">
          {/* Hero Section */}
          <section className="lp-hero">
            <div className="lp-hero-bg"></div>
            <div className="lp-hero-inner">
              <div className="lp-hero-content">
                <span className="lp-hero-badge">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
                    <polyline points="9 22 9 12 15 12 15 22"/>
                  </svg>
                  新しい賃貸検索
                </span>
                <h1 className="lp-hero-title">
                  <span className="lp-hero-title-accent">気になる物件だけ</span>が<br />残る部屋探し
                </h1>
                <p className="lp-hero-subtitle">
                  条件に合わない物件をワンタップで非表示に。<br />
                  検索するたびに候補が絞られていく、新しい体験です。
                </p>
                <p className="lp-hero-note">
                  ※掲載料で上位が決まる検索ではなく、ユーザーの判断が中心の設計です。
                </p>
                <button className="lp-hero-cta" onClick={() => {
                  document.querySelector('.lp-cta')?.scrollIntoView({ behavior: 'smooth' });
                }}>
                  検索をはじめる
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M17 8l4 4m0 0l-4 4m4-4H3" />
                  </svg>
                </button>
              </div>
            </div>
          </section>

          {/* Problem Section */}
          <section className="lp-problem">
            <div className="lp-problem-inner">
              <p className="lp-section-label">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="8" x2="12" y2="12"/>
                  <line x1="12" y1="16" x2="12.01" y2="16"/>
                </svg>
                こんな経験ありませんか？
              </p>
              <h2 className="lp-problem-title">
                部屋探しが疲れるのは、<br />
                情報が多すぎるから
              </h2>
              <div className="lp-problem-list">
                <div className="lp-problem-item">
                  <div className="lp-problem-icon">
                    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                      <polyline points="14 2 14 8 20 8"/>
                      <line x1="16" y1="13" x2="8" y2="13"/>
                      <line x1="16" y1="17" x2="8" y2="17"/>
                      <polyline points="10 9 9 9 8 9"/>
                    </svg>
                  </div>
                  <p className="lp-problem-text">条件を絞っても、関係ない物件が大量に表示される</p>
                </div>
                <div className="lp-problem-item">
                  <div className="lp-problem-icon">
                    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <polyline points="23 4 23 10 17 10"/>
                      <polyline points="1 20 1 14 7 14"/>
                      <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                    </svg>
                  </div>
                  <p className="lp-problem-text">一度見て却下した物件が、検索するたびに出てくる</p>
                </div>
                <div className="lp-problem-item">
                  <div className="lp-problem-icon">
                    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <circle cx="12" cy="12" r="10"/>
                      <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/>
                      <line x1="12" y1="17" x2="12.01" y2="17"/>
                    </svg>
                  </div>
                  <p className="lp-problem-text">表記が正しいか分からず、すべて疑いながら見る必要がある</p>
                </div>
                <div className="lp-problem-item">
                  <div className="lp-problem-icon">
                    <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <line x1="8" y1="6" x2="21" y2="6"/>
                      <line x1="8" y1="12" x2="21" y2="12"/>
                      <line x1="8" y1="18" x2="21" y2="18"/>
                      <line x1="3" y1="6" x2="3.01" y2="6"/>
                      <line x1="3" y1="12" x2="3.01" y2="12"/>
                      <line x1="3" y1="18" x2="3.01" y2="18"/>
                    </svg>
                  </div>
                  <p className="lp-problem-text">探せば探すほど、どれがいいのか分からなくなる</p>
                </div>
              </div>
            </div>
          </section>

          {/* Solution Section */}
          <section className="lp-solution">
            <div className="lp-solution-inner">
              <p className="lp-section-label">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
                </svg>
                shiboroomの特徴
              </p>
              <div className="lp-solution-header">
                <p className="lp-solution-tagline">「増やす」より「減らす」</p>
                <h2 className="lp-solution-title">
                  興味のない物件を消していく、<br />
                  新しい検索体験
                </h2>
              </div>

              <div className="lp-features">
                <div className="lp-feature">
                  <div className="lp-feature-header">
                    <div className="lp-feature-number">1</div>
                    <h3 className="lp-feature-title">ワンタップで非表示<span className="lp-feature-badge">コア</span></h3>
                  </div>
                  <p className="lp-feature-desc">
                    気に入らない物件はその場で非表示に。一度消した物件は、次回から表示されません。使うほど検索結果がスッキリしていきます。
                  </p>
                  <div className="lp-feature-visual">
                    <div className="lp-demo-card">
                      <div className="lp-demo-card-info">
                        <p className="lp-demo-card-title">渋谷区恵比寿 1K</p>
                        <p className="lp-demo-card-meta">8.5万円 / 徒歩7分 / 25㎡</p>
                      </div>
                      <button className="lp-demo-hide-btn" title="非表示にする">
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>

                <div className="lp-feature">
                  <div className="lp-feature-header">
                    <div className="lp-feature-number">2</div>
                    <h3 className="lp-feature-title">◎ △ × で絞り込み<span className="lp-feature-badge">コア</span></h3>
                  </div>
                  <p className="lp-feature-desc">
                    「できれば8分以内、10分までなら許容」のような曖昧な条件を設定できます。×は自動で除外、△は注意つきで表示します。
                  </p>
                  <div className="lp-feature-visual">
                    <p className="lp-demo-rating-label">駅からの距離</p>
                    <div className="lp-demo-rating">
                      <div className="lp-rating-option">
                        <button className="lp-rating-btn active-good">◎</button>
                        <span className="lp-rating-label">〜8分</span>
                      </div>
                      <div className="lp-rating-option">
                        <button className="lp-rating-btn active-ok">△</button>
                        <span className="lp-rating-label">9〜10分</span>
                      </div>
                      <div className="lp-rating-option">
                        <button className="lp-rating-btn active-bad">×</button>
                        <span className="lp-rating-label">11分〜</span>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="lp-feature">
                  <div className="lp-feature-header">
                    <div className="lp-feature-number">3</div>
                    <h3 className="lp-feature-title">怪しい物件が分かる<span className="lp-feature-badge">コア</span></h3>
                  </div>
                  <p className="lp-feature-desc">
                    ユーザーからの「表記と違うかも」報告を可視化します。運営が正否を断定するのではなく、報告があること・件数を表示します。すべての物件を疑って見る必要がなくなります。
                  </p>
                  <div className="lp-feature-visual">
                    <div className="lp-demo-report">
                      <div className="lp-report-header">
                        <svg className="lp-report-icon" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                          <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                        </svg>
                        <span className="lp-report-title">相違報告あり</span>
                      </div>
                      <div className="lp-report-item">
                        <span>独立洗面台の表記が怪しい</span>
                        <span className="lp-report-count">3件</span>
                      </div>
                      <div className="lp-report-item">
                        <span>写真と実際の印象が違う</span>
                        <span className="lp-report-count">2件</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* Plus Alpha */}
              <div className="lp-plus-alpha">
                <p className="lp-plus-alpha-header">＋α あると便利</p>
                <div className="lp-plus-alpha-list">
                  <div className="lp-plus-alpha-item">
                    <span className="lp-plus-alpha-icon">
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                        <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                      </svg>
                    </span>
                    <span><strong>メモ機能</strong>:却下理由・内見メモを残せる</span>
                  </div>
                  <div className="lp-plus-alpha-item">
                    <span className="lp-plus-alpha-icon">
                      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                        <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                        <line x1="3" y1="9" x2="21" y2="9"/>
                        <line x1="9" y1="21" x2="9" y2="9"/>
                      </svg>
                    </span>
                    <span><strong>広さの不明可視化</strong>:LDKなど定義が不明な場合は「不明」として扱える(比較しやすい)</span>
                  </div>
                </div>
              </div>
            </div>
          </section>

          {/* Why Section */}
          <section className="lp-why">
            <div className="lp-why-inner">
              <p className="lp-section-label">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="16" x2="12" y2="12"/>
                  <line x1="12" y1="8" x2="12.01" y2="8"/>
                </svg>
                なぜ他ではできなかった?
              </p>
              <h2 className="lp-why-title">掲載料モデルの限界</h2>
              <div className="lp-why-text">
                <p>
                  既存の不動産ポータルサイトは、不動産会社から掲載料をもらうモデルが中心です。そのため、ユーザーが物件をどんどん非表示にする体験は、構造的に提供しづらいことがあります。
                </p>
                <p>
                  結果として、興味のない物件が何度も表示され、ユーザーは「使いにくい」と感じながらも我慢して使い続けていました。
                </p>
              </div>
              <div className="lp-why-highlight">
                <p className="lp-why-highlight-text">
                  shiboroomは、探す量を減らすことを最優先に、判断しやすい検索体験を提供します。
                </p>
              </div>
            </div>
          </section>

          {/* CTA Section */}
          <section className="lp-cta">
            <div className="lp-cta-content">
              <span className="lp-cta-badge">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="20 6 9 17 4 12"/>
                </svg>
                登録不要で今すぐ検索できます
              </span>
              <h2 className="lp-cta-title">部屋探しをはじめる</h2>
              <div className="lp-cta-buttons">
                <button className="lp-cta-btn" onClick={() => setCurrentView('results')}>
                  検索をはじめる
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M17 8l4 4m0 0l-4 4m4-4H3" />
                  </svg>
                </button>
              </div>
            </div>
          </section>
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
                  onHide={() => hideProperty(property.id)}
                  memo={propertyMemos[property.id] || ''}
                  onMemoChange={(memo) => setPropertyMemo(property.id, memo)}
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
  onStatusChange,
  onHide,
  memo,
  onMemoChange
}: {
  property: Property
  formatRent: (rent?: number) => string | null
  status: PropertyStatus
  onStatusChange: (status: PropertyStatus) => void
  onHide: () => void
  memo: string
  onMemoChange: (memo: string) => void
}) {
  const [imageError, setImageError] = useState(false)
  const [showMemoInput, setShowMemoInput] = useState(false)
  const [memoText, setMemoText] = useState(memo)
  const router = useRouter()

  // Sync local state when prop changes
  useEffect(() => {
    setMemoText(memo)
  }, [memo])

  const handleMemoSave = () => {
    onMemoChange(memoText)
    setShowMemoInput(false)
  }

  const handleMemoCancel = () => {
    setMemoText(memo)
    setShowMemoInput(false)
  }

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
        <button
          className="hide-property-btn"
          onClick={(e) => {
            e.stopPropagation()
            onHide()
          }}
          title="非表示にする"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M18 6 6 18"/>
            <path d="m6 6 12 12"/>
          </svg>
        </button>
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

        {/* Memo Section */}
        <div className="property-memo-section">
          {!showMemoInput && !memo && (
            <button
              className="memo-add-btn"
              onClick={(e) => {
                e.stopPropagation()
                setShowMemoInput(true)
              }}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
              </svg>
              メモを追加
            </button>
          )}

          {!showMemoInput && memo && (
            <div className="property-memo-display" onClick={(e) => {
              e.stopPropagation()
              setShowMemoInput(true)
            }}>
              <div className="memo-display-header">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                  <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                </svg>
                <span>メモ</span>
              </div>
              <p className="memo-display-text">{memo}</p>
            </div>
          )}

          {showMemoInput && (
            <div className="property-memo-input" onClick={(e) => e.stopPropagation()}>
              <textarea
                className="memo-textarea"
                value={memoText}
                onChange={(e) => setMemoText(e.target.value)}
                placeholder="メモを入力（却下理由・内見メモなど）"
                rows={3}
                autoFocus
              />
              <div className="memo-actions">
                <button className="memo-save-btn" onClick={handleMemoSave}>
                  保存
                </button>
                <button className="memo-cancel-btn" onClick={handleMemoCancel}>
                  キャンセル
                </button>
              </div>
            </div>
          )}
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
