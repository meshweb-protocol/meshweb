import { useState, useEffect, useRef } from 'react';
import { StartNewNetwork, ConnectToNetwork, GetDashboardStats, ToggleOfferResources, LoadIdentity, GenerateIdentity, RestoreIdentity, GetPublicKey, ExportIdentity, SelectFile, UploadFile, DownloadFile, GenerateShareLink, GenerateMeshwebFile, GetMyFiles, DeleteFile, RegisterFileAssociation, GetStartupFile, FindAvailableNodes, StartRental, StopRental, GetRentalStatus, GetDownloadedFiles, OpenFile } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';
import './App.css';
import en from './locales/en.json';
import uz from './locales/uz.json';
import ru from './locales/ru.json';

const translations: Record<string, Record<string, string>> = { en, uz, ru };

function App() {
  const [lang, setLang] = useState(localStorage.getItem('meshweb_lang') || 'en');
  const t = (key: string) => translations[lang]?.[key] || key;
  const changeLang = (l: string) => { setLang(l); localStorage.setItem('meshweb_lang', l); };
  const [showSettings, setShowSettings] = useState(false);
  const [showLogs, setShowLogs] = useState(false);
  const [connected, setConnected] = useState(false);
  const [inviteLink, setInviteLink] = useState('');
  const [inputLink, setInputLink] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  
  const [activeTab, setActiveTab] = useState<'dashboard' | 'storage'>('dashboard');
  const [storageSubTab, setStorageSubTab] = useState<'uploads' | 'downloads'>('uploads');
  const [myFiles, setMyFiles] = useState<any[]>([]);
  const [downloadedFiles, setDownloadedFiles] = useState<any[]>([]);
  const [downloadLink, setDownloadLink] = useState('');
  const [downloadProgress, setDownloadProgress] = useState(-1);
  const [showDownloadModal, setShowDownloadModal] = useState(false);
  const [isDragOver, setIsDragOver] = useState(false);
  
  const [showRentModal, setShowRentModal] = useState(false);
  const [showComputeModal, setShowComputeModal] = useState(false);
  const [rentDuration, setRentDuration] = useState(1);
  const [rentCost, setRentCost] = useState(0);
  const [rentStep, setRentStep] = useState<'form' | 'finding' | 'active'>('form');
  const [activeRentalJobId, setActiveRentalJobId] = useState('');
  const [activeRentalStats, setActiveRentalStats] = useState<any>(null);
  
  const [identityLoaded, setIdentityLoaded] = useState<boolean | null>(null);
  const [onboardingView, setOnboardingView] = useState<'main' | 'create' | 'restore'>('main');
  const [seedPhrase, setSeedPhrase] = useState('');
  const [inputSeed, setInputSeed] = useState('');
  const [savedSeed, setSavedSeed] = useState(false);
  const [myPublicKey, setMyPublicKey] = useState('');

  const [stats, setStats] = useState({
    peerId: '',
    inviteLink: '',
    balance: 0,
    todayIncome: 0,
    totalIncome: 0,
    cpu: 0,
    ram: 0,
    connectedPeers: 0,
    activeJobs: 0,
    isBuyer: false,
  });

  const [offerResources, setOfferResources] = useState(true);
  const [logs, setLogs] = useState<string[]>([]);
  const logsEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Load Identity on mount
    LoadIdentity().then((loaded: boolean) => {
      setIdentityLoaded(loaded);
      if (loaded) {
        GetPublicKey().then(setMyPublicKey);
      }
    });

    // Listen to backend events
    EventsOn("activity-log", (msg: string) => {
      setLogs((prev) => [...prev, msg]);
    });
    EventsOn("download-progress", (progress: number) => {
      setDownloadProgress(progress);
    });

    // Check for startup file (double-click association)
    GetStartupFile().then((fileOrLink: string) => {
      if (fileOrLink) {
        setDownloadLink(fileOrLink);
        setActiveTab('storage');
      }
    });

    // Poll stats every 2 seconds
    const interval = setInterval(() => {
      if (connected) {
        GetDashboardStats().then((data: any) => {
          setStats(data);
        });
      }
    }, 2000);

    return () => clearInterval(interval);
  }, [connected]);

  useEffect(() => {
    // Auto-scroll logs
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [logs]);

  const handleStartNetwork = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await StartNewNetwork();
      if (res.success) {
        setConnected(true);
      } else {
        setError(res.error || 'Failed to start network');
      }
    } catch (e: any) {
      setError(e.toString());
    }
    setLoading(false);
  };

  const handleConnect = async () => {
    if (!inputLink) return;
    setLoading(true);
    setError('');
    try {
      const res = await ConnectToNetwork(inputLink);
      if (res.success) {
        setConnected(true);
      } else {
        setError(res.error || 'Failed to connect');
      }
    } catch (e: any) {
      setError(e.toString());
    }
    setLoading(false);
  };

  const handleToggleOffer = (val: boolean) => {
    setOfferResources(val);
    ToggleOfferResources(val);
  };

  const handleCreateIdentity = async () => {
    setLoading(true);
    const res = await GenerateIdentity();
    if (res.success) {
      setSeedPhrase(res.seedPhrase);
      setOnboardingView('create');
    }
    setLoading(false);
  };

  const handleFinishCreate = async () => {
    if (!savedSeed) return;
    setIdentityLoaded(true);
    GetPublicKey().then(setMyPublicKey);
  };

  const handleRestoreIdentity = async () => {
    if (!inputSeed) return;
    setLoading(true);
    const success = await RestoreIdentity(inputSeed.trim());
    if (success) {
      setIdentityLoaded(true);
      GetPublicKey().then(setMyPublicKey);
    } else {
      setError('Invalid seed phrase');
    }
    setLoading(false);
  };

  const loadMyFiles = async () => {
    const files = await GetMyFiles();
    setMyFiles(files || []);
    const downloaded = await GetDownloadedFiles();
    setDownloadedFiles(downloaded || []);
  };

  useEffect(() => {
    if (activeTab === 'storage') {
      loadMyFiles();
    }
  }, [activeTab]);

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
  };

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
    
    // Wails does not support accessing absolute path from HTML5 DataTransfer yet by default, 
    // but WebView2 on Windows might support e.dataTransfer.files[0].path
    // If not, we will fallback to SelectFile() manually or alert the user.
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      const file: any = e.dataTransfer.files[0];
      const path = file.path || file.name; // file.path works in Electron/WebView2 if enabled
      if (path && path.includes('\\')) {
        setLoading(true);
        const res = await UploadFile(path);
        if (res.success) loadMyFiles();
        else alert("Upload failed: " + res.error);
        setLoading(false);
      } else {
        // Fallback if path is not available
        handleUpload();
      }
    }
  };

  const handleUpload = async () => {
    const path = await SelectFile();
    if (!path) return;
    setLoading(true);
    const res = await UploadFile(path);
    if (res.success) {
      loadMyFiles();
    } else {
      alert("Upload failed: " + res.error);
    }
    setLoading(false);
  };

  const handleDownload = async () => {
    if (!downloadLink) return;
    setLoading(true);
    setDownloadProgress(0);
    const res = await DownloadFile(downloadLink);
    if (res.success) {
      alert(t('Download') + " ✅ " + res.path);
      loadMyFiles();
    } else {
      alert("Error: " + res.error);
    }
    setDownloadProgress(-1);
    setLoading(false);
    setDownloadLink('');
  };

  const handleShareLink = async (fileId: string) => {
    const link = await GenerateShareLink(fileId);
    if (link) {
      navigator.clipboard.writeText(link);
      alert(t('Link copied!'));
    }
  };

  const handleExportMeshweb = async (fileId: string) => {
    const res = await GenerateMeshwebFile(fileId);
    if (res.success) {
      const blob = new Blob([res.content], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${fileId}.meshweb`;
      a.click();
    }
  };

  const handleDeleteFile = async (fileId: string) => {
    await DeleteFile(fileId);
    loadMyFiles();
  };

  const handleRegisterAssoc = async () => {
    await RegisterFileAssociation();
    alert("Windows File Association Registered ✅");
  };

  useEffect(() => {
    // 4 CPU * 0.1 + 8 RAM * 0.05 = 0.8 / hour
    setRentCost(0.8 * rentDuration);
  }, [rentDuration]);

  const handleStartRent = async () => {
    setRentStep('finding');
    // Hardcoded request: 4 CPU, 8GB RAM, no GPU
    const res = await FindAvailableNodes(4, 8, false);
    if (res.success && res.nodes && res.nodes.length > 0) {
      const best = res.nodes.sort((a: any, b: any) => a.latency - b.latency)[0];
      setTimeout(async () => {
        const startRes = await StartRental(best.peer_id, 4, 8, false, rentDuration);
        if (startRes.success) {
          setActiveRentalJobId(startRes.jobId);
          setRentStep('active');
        } else {
          alert("Failed to rent: " + startRes.error);
          setRentStep('form');
        }
      }, 1500);
    } else {
      alert("No suitable nodes available.");
      setRentStep('form');
    }
  };

  const handleStopRent = async () => {
    if (activeRentalJobId) {
      await StopRental(activeRentalJobId);
      setActiveRentalJobId('');
      setShowRentModal(false);
      setRentStep('form');
    }
  };

  useEffect(() => {
    let intv: any;
    if (rentStep === 'active' && activeRentalJobId) {
      intv = setInterval(async () => {
        const s = await GetRentalStatus(activeRentalJobId);
        if (s.success && s.job.is_active) {
          setActiveRentalStats(s);
        } else {
          setActiveRentalJobId('');
          setRentStep('form');
          setShowRentModal(false);
        }
      }, 2000);
    }
    return () => clearInterval(intv);
  }, [rentStep, activeRentalJobId]);

  if (identityLoaded === null) {
    return <div className="h-screen flex items-center justify-center bg-gray-900 text-white font-mono">Loading...</div>;
  }

  if (identityLoaded === false) {
    return (
      <div className="min-h-screen bg-gray-900 flex flex-col items-center justify-center text-gray-100 font-sans p-6">
        <div className="w-full max-w-md flex flex-col items-center space-y-8 animate-fade-in-up">
          <div className="flex flex-col items-center space-y-4">
            <div className="w-24 h-24 bg-primary rounded-3xl flex items-center justify-center shadow-lg shadow-primary/30 rotate-12 hover:rotate-0 transition-transform duration-500">
              <svg className="w-12 h-12 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <h1 className="text-4xl font-extrabold tracking-tight">Mesh<span className="text-primary">web</span></h1>
            <p className="text-gray-400 text-center">{t('Welcome to Meshweb')}</p>
          </div>

          <div className="w-full bg-gray-800 p-8 rounded-2xl shadow-2xl space-y-6 border border-gray-700">
            {error && <div className="text-danger text-sm text-center bg-danger/10 p-2 rounded">{error}</div>}
            
            {onboardingView === 'main' && (
              <div className="space-y-4">
                <button
                  onClick={handleCreateIdentity}
                  disabled={loading}
                  className="w-full py-3 px-4 bg-primary hover:bg-primary/90 text-white rounded-xl font-medium transition-all shadow-lg"
                >
                  {t('Create New Account')}
                </button>
                <div className="relative flex items-center py-2">
                  <div className="flex-grow border-t border-gray-700"></div>
                  <span className="flex-shrink-0 mx-4 text-gray-500 text-sm">{t('YOKI')}</span>
                  <div className="flex-grow border-t border-gray-700"></div>
                </div>
                <button
                  onClick={() => setOnboardingView('restore')}
                  className="w-full py-3 px-4 bg-gray-700 hover:bg-gray-600 text-white rounded-xl font-medium transition-all"
                >
                  {t('Restore Account')}
                </button>
              </div>
            )}

            {onboardingView === 'create' && (
              <div className="space-y-6">
                <h3 className="text-white font-bold">{t('Your Seed Phrase')}</h3>
                <div className="bg-gray-900 p-4 rounded-xl font-mono text-sm text-primary select-all break-words border border-primary/20">
                  {seedPhrase}
                </div>
                <label className="flex items-center space-x-3 cursor-pointer">
                  <input type="checkbox" checked={savedSeed} onChange={(e) => setSavedSeed(e.target.checked)} className="rounded text-primary focus:ring-primary bg-gray-900 border-gray-700" />
                  <span className="text-sm text-gray-300">{t('I have saved my seed phrase')}</span>
                </label>
                <button
                  onClick={handleFinishCreate}
                  disabled={!savedSeed}
                  className="w-full py-3 px-4 bg-primary hover:bg-primary/90 text-white rounded-xl font-medium transition-all disabled:opacity-50"
                >
                  Davom etish
                </button>
              </div>
            )}

            {onboardingView === 'restore' && (
              <div className="space-y-4">
                <h3 className="text-white font-bold">{t('Restore Account')}</h3>
                <textarea
                  value={inputSeed}
                  onChange={(e) => setInputSeed(e.target.value)}
                  placeholder="word1 word2 word3..."
                  className="w-full h-24 bg-gray-900 border border-gray-700 text-white p-3 rounded-xl outline-none focus:border-primary"
                />
                <button
                  onClick={handleRestoreIdentity}
                  disabled={loading || !inputSeed}
                  className="w-full py-3 px-4 bg-primary hover:bg-primary/90 text-white rounded-xl font-medium transition-all disabled:opacity-50"
                >
                  Davom etish
                </button>
                <button onClick={() => setOnboardingView('main')} className="w-full text-sm text-gray-400 hover:text-white">Orqaga</button>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  if (!connected) {
    return (
      <div className="min-h-screen bg-gray-900 flex flex-col items-center justify-center text-gray-100 font-sans p-6">
        <div className="w-full max-w-md flex flex-col items-center space-y-8 animate-fade-in-up">
          {/* Logo Section */}
          <div className="flex flex-col items-center space-y-4">
            <div className="w-24 h-24 bg-primary rounded-3xl flex items-center justify-center shadow-lg shadow-primary/30 rotate-12 hover:rotate-0 transition-transform duration-500">
              <svg className="w-12 h-12 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <h1 className="text-4xl font-extrabold tracking-tight">Mesh<span className="text-primary">web</span></h1>
            <p className="text-gray-400 text-center">Decentralized Compute Network</p>
          </div>

          {/* Action Cards */}
          <div className="w-full bg-gray-800 p-8 rounded-2xl shadow-2xl border border-gray-700 flex flex-col items-center">
            {error && <div className="text-danger text-sm text-center bg-danger/10 p-2 rounded w-full mb-4">{error}</div>}
            
            <button
              onClick={handleStartNetwork}
              disabled={loading}
              className="w-full py-4 px-4 bg-primary hover:bg-primary/90 text-white rounded-2xl font-bold text-lg transition-all transform hover:scale-[1.02] active:scale-95 shadow-lg shadow-primary/20 disabled:opacity-50"
            >
              {loading ? t('Kutish...') : t('Join Meshweb')}
            </button>
            <p className="text-gray-500 text-xs mt-4 text-center">
              Decentralized resources powered by community.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="h-screen bg-gray-900 text-gray-100 font-sans p-6 flex flex-col space-y-6 overflow-hidden">
      {/* Top Bar */}
      <header className="flex justify-between items-center bg-gray-800 p-4 rounded-2xl shadow-lg border border-gray-700">
        <div className="flex items-center space-x-3 cursor-pointer group" onClick={() => setActiveTab('dashboard')}>
          <div className="w-10 h-10 bg-primary rounded-xl flex items-center justify-center group-hover:scale-105 transition-transform">
            <svg className="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
          <h2 className="text-xl font-bold group-hover:text-primary transition-colors">Meshweb</h2>
        </div>
        <div className="flex items-center space-x-6">
          <div className="flex items-center space-x-2">
            <span className="relative flex h-3 w-3">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-success opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-success"></span>
            </span>
            <span className="text-sm font-medium text-success">{t('Connected')}</span>
          </div>
          <div className="flex items-center space-x-3 bg-gray-900 px-3 py-2 rounded-xl border border-gray-700 group cursor-pointer" title={myPublicKey}>
            <div 
              className="w-8 h-8 rounded-full shadow-inner flex items-center justify-center text-xs font-bold text-white uppercase"
              style={{ backgroundColor: `#${myPublicKey.substring(myPublicKey.length - 6)}` }}
            >
              {myPublicKey.substring(0, 2)}
            </div>
            <div className="flex flex-col">
              <span className="text-xs text-gray-500 uppercase tracking-wider">{t('Identity')}</span>
              <span className="text-sm font-mono text-gray-200">MW-{myPublicKey.substring(myPublicKey.length - 8)}</span>
            </div>
          </div>
        </div>
      </header>

      {activeTab === 'dashboard' && (
        <div className="flex-grow flex flex-col justify-center items-center space-y-8 min-h-0 animate-fade-in-up">
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 w-full max-w-4xl">
            {/* Balance Card */}
            <div className="bg-gray-800 p-10 rounded-[2rem] shadow-2xl border border-gray-700 flex flex-col items-center justify-center transform transition-transform hover:scale-[1.02]">
              <span className="text-gray-400 text-sm font-bold uppercase tracking-widest mb-4">{t('MENING HISOBIM')}</span>
              <div className="flex items-baseline space-x-2 mb-6">
                <span className="text-6xl font-extrabold text-white">{stats.balance.toFixed(2)}</span>
                <span className="text-xl text-primary font-bold">MWC</span>
              </div>
              <div className="bg-gray-900/50 px-6 py-3 rounded-full border border-gray-700">
                <span className="text-gray-400 mr-2">{t('Bugun')}:</span>
                <span className="text-success font-bold">+{stats.todayIncome.toFixed(2)} MWC</span>
              </div>

              <button 
                onClick={() => setShowComputeModal(true)} 
                className="mt-8 flex items-center space-x-1 text-sm text-gray-500 hover:text-gray-300 transition-colors opacity-70 hover:opacity-100"
              >
                <span>⚡</span>
                <span>Compute <span className="text-xs bg-gray-700 text-gray-400 px-1.5 py-0.5 rounded-full ml-1">Coming Soon</span></span>
              </button>
            </div>

            {/* Network Card */}
            <div className="bg-gray-800 p-10 rounded-[2rem] shadow-2xl border border-gray-700 flex flex-col items-center justify-center transform transition-transform hover:scale-[1.02]">
              <span className="text-gray-400 text-sm font-bold uppercase tracking-widest mb-4">{t('TARMOQ')}</span>
              <div className="flex items-baseline space-x-2 mb-6">
                <span className="text-6xl font-extrabold text-white">{stats.connectedPeers}</span>
                <span className="text-xl text-gray-400 font-medium">{t('node ulangan')}</span>
              </div>
              <div className="bg-gray-900/50 px-6 py-3 rounded-full border border-gray-700">
                <span className="text-gray-400 mr-2">{t('Tezlik')}:</span>
                <span className="text-white font-bold">~45ms</span>
              </div>
            </div>
          </div>

        </div>
      )}

      {activeTab === 'storage' && (
        <div 
          className="flex-grow flex flex-col p-6 animate-fade-in-up"
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
        >
          <div className={`w-full bg-[#1e2329] rounded-[1rem] shadow-2xl border flex flex-col h-full overflow-hidden transition-colors ${isDragOver ? 'border-success bg-[#1e2329]/90' : 'border-gray-700'}`}>
            
            {/* Top Bar with Tabs */}
            <div className="flex justify-between items-center p-4 border-b border-gray-700 bg-[#161a1e]">
              <div className="flex space-x-6 items-center">
                <h2 className="text-xl font-bold text-white flex items-center space-x-2 mr-4">
                  <span className="text-2xl">📂</span>
                </h2>
                
                <button 
                  onClick={() => setStorageSubTab('uploads')}
                  className={`text-sm font-bold pb-1 transition-colors ${storageSubTab === 'uploads' ? 'text-primary border-b-2 border-primary' : 'text-gray-500 hover:text-gray-300'}`}
                >
                  📤 {t('My Uploads')}
                </button>
                <button 
                  onClick={() => setStorageSubTab('downloads')}
                  className={`text-sm font-bold pb-1 transition-colors ${storageSubTab === 'downloads' ? 'text-primary border-b-2 border-primary' : 'text-gray-500 hover:text-gray-300'}`}
                >
                  📥 {t('Downloaded')}
                </button>
              </div>

              <div className="flex space-x-3">
                <button 
                  onClick={() => setShowDownloadModal(true)}
                  className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg font-medium text-sm transition-all flex items-center space-x-2"
                >
                  <span>⬇</span>
                  <span>{t('Link orqali yuklab olish') || 'Download via Link'}</span>
                </button>
                <button 
                  onClick={handleUpload}
                  disabled={loading}
                  className="px-4 py-2 bg-primary hover:bg-primary/90 text-white rounded-lg font-bold text-sm transition-all shadow-lg shadow-primary/20 flex items-center space-x-2 disabled:opacity-50"
                >
                  <span>+</span>
                  <span>{t('Fayl yuklash') || 'Upload'}</span>
                </button>
              </div>
            </div>

            {/* Drag Overlay Bar if files exist */}
            {isDragOver && (myFiles.length > 0 || storageSubTab === 'downloads') && (
              <div className="bg-success/20 border-b border-success text-success text-center py-2 text-sm font-bold">
                {t('Yangi fayl yuklash uchun tashlang') || 'Drop files here to upload'}
              </div>
            )}

            {/* Content */}
            <div className="flex-grow overflow-y-auto custom-scrollbar">
              
              {/* UPLOADS TAB */}
              {storageSubTab === 'uploads' && (
                <>
                  {myFiles.length === 0 && downloadProgress < 0 ? (
                    <div className="flex flex-col items-center justify-center h-full space-y-4 text-gray-500 border-2 border-dashed border-gray-700 m-6 rounded-2xl p-12 transition-colors hover:border-gray-500">
                      <span className="text-6xl mb-2 opacity-50">📂</span>
                      <p className="text-lg font-medium text-white">{t('Drop files here')}</p>
                      <p className="text-sm opacity-70">{t('or')}</p>
                      <button 
                        onClick={handleUpload}
                        disabled={loading}
                        className="mt-4 py-3 px-8 bg-primary hover:bg-primary/90 text-white rounded-xl font-bold shadow-lg shadow-primary/20 transition-transform transform hover:scale-105 active:scale-95 disabled:opacity-50"
                      >
                        + {t('Fayl yuklash') || 'Upload File'}
                      </button>
                    </div>
                  ) : (
                    <table className="w-full text-left border-collapse">
                      <thead className="bg-[#1a1f24] sticky top-0 z-10 border-b border-gray-700 text-xs uppercase text-gray-400">
                        <tr>
                          <th className="py-3 px-4 w-12 text-center"></th>
                          <th className="py-3 px-4 font-medium">{t('Name')}</th>
                          <th className="py-3 px-4 font-medium w-24">{t('Size')}</th>
                          <th className="py-3 px-4 font-medium w-32">{t('Status')}</th>
                          <th className="py-3 px-4 font-medium w-40 text-right">{t('Actions')}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-800 text-sm">
                        {downloadProgress >= 0 && (
                          <tr className="hover:bg-gray-800/50 transition-colors bg-primary/5">
                            <td className="py-3 px-4 text-center text-xl">⬇</td>
                            <td className="py-3 px-4 font-medium text-white max-w-[200px] truncate" title={downloadLink}>
                              Yuklanmoqda...
                            </td>
                            <td className="py-3 px-4 text-gray-400">--</td>
                            <td className="py-3 px-4">
                              <div className="flex flex-col space-y-1">
                                <span className="text-xs text-primary font-bold">{Math.floor(downloadProgress)}%</span>
                                <div className="w-full bg-gray-800 rounded-full h-1.5 overflow-hidden">
                                  <div className="bg-primary h-full transition-all duration-300" style={{ width: `${downloadProgress}%` }}></div>
                                </div>
                              </div>
                            </td>
                            <td className="py-3 px-4 text-right">
                              <button className="p-1.5 text-gray-500 hover:text-white rounded-md transition-colors" disabled>⏳</button>
                            </td>
                          </tr>
                        )}
                        
                        {myFiles.map((f, i) => {
                          const ext = f.file_name.split('.').pop()?.toLowerCase();
                          let icon = '📄';
                          if (['mp4', 'avi', 'mkv', 'mov'].includes(ext)) icon = '📹';
                          else if (['mp3', 'wav', 'ogg'].includes(ext)) icon = '🎵';
                          else if (['jpg', 'jpeg', 'png', 'gif', 'webp'].includes(ext)) icon = '🖼';
                          else if (['zip', 'rar', 'tar', 'gz'].includes(ext)) icon = '📦';
                          
                          let sizeStr = '';
                          if (f.file_size < 1024) sizeStr = f.file_size + ' B';
                          else if (f.file_size < 1024*1024) sizeStr = (f.file_size/1024).toFixed(1) + ' KB';
                          else if (f.file_size < 1024*1024*1024) sizeStr = (f.file_size/1024/1024).toFixed(1) + ' MB';
                          else sizeStr = (f.file_size/1024/1024/1024).toFixed(2) + ' GB';

                          const isComplete = f.file_name.startsWith('downloaded_') || f.creator_id !== stats?.peerId;
                          const stateBadge = isComplete ? (
                            <div className="flex items-center space-x-1.5">
                              <div className="w-2 h-2 rounded-full bg-primary animate-pulse"></div>
                              <span className="text-primary text-xs font-bold">{t('Complete') || 'Complete'}</span>
                            </div>
                          ) : (
                            <div className="flex items-center space-x-1.5">
                              <div className="w-2 h-2 rounded-full bg-success animate-pulse"></div>
                              <span className="text-success text-xs font-bold">{t('Seeding') || 'Seeding'}</span>
                            </div>
                          );

                          return (
                            <tr key={i} className="hover:bg-gray-800/50 transition-colors group">
                              <td className="py-3 px-4 text-center text-xl">{icon}</td>
                              <td className="py-3 px-4 font-medium text-gray-200 max-w-[200px] truncate" title={f.file_name}>{f.file_name}</td>
                              <td className="py-3 px-4 text-gray-400">{sizeStr}</td>
                              <td className="py-3 px-4">{stateBadge}</td>
                              <td className="py-3 px-4 text-right opacity-50 group-hover:opacity-100 transition-opacity">
                                <div className="flex justify-end space-x-1 items-center">
                                  {f.local_path && (
                                    <button onClick={() => OpenFile(f.local_path)} className="px-3 py-1.5 text-xs font-bold text-gray-300 hover:text-white bg-gray-700 hover:bg-gray-600 rounded-md transition-colors flex items-center space-x-1" title={t('Ochish') || 'Open'}>
                                      <span>📂</span>
                                      <span>{t('Open')}</span>
                                    </button>
                                  )}
                                  <button onClick={() => handleShareLink(f.file_id)} className="p-1.5 text-gray-400 hover:text-primary hover:bg-primary/10 rounded-md transition-colors" title={t('Ulashish')}>🔗</button>
                                  <button onClick={() => handleDeleteFile(f.file_id)} className="p-1.5 text-gray-400 hover:text-danger hover:bg-danger/10 rounded-md transition-colors" title="Delete">🗑️</button>
                                </div>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  )}
                </>
              )}

              {/* DOWNLOADS TAB */}
              {storageSubTab === 'downloads' && (
                <>
                  {downloadedFiles.length === 0 ? (
                    <div className="flex flex-col items-center justify-center h-full space-y-4 text-gray-500">
                      <span className="text-6xl mb-2 opacity-50">📥</span>
                      <p className="text-lg">{t('No files yet')}</p>
                    </div>
                  ) : (
                    <table className="w-full text-left border-collapse">
                      <thead className="bg-[#1a1f24] sticky top-0 z-10 border-b border-gray-700 text-xs uppercase text-gray-400">
                        <tr>
                          <th className="py-3 px-4 w-12 text-center"></th>
                          <th className="py-3 px-4 font-medium">{t('Name')}</th>
                          <th className="py-3 px-4 font-medium w-24">{t('Size')}</th>
                          <th className="py-3 px-4 font-medium w-40">{t('Date')}</th>
                          <th className="py-3 px-4 font-medium w-32 text-right">{t('Actions')}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-800 text-sm">
                        {downloadedFiles.map((df, i) => {
                          const ext = df.file_name.split('.').pop()?.toLowerCase();
                          let icon = '📄';
                          if (['mp4', 'avi', 'mkv', 'mov'].includes(ext)) icon = '📹';
                          else if (['mp3', 'wav', 'ogg'].includes(ext)) icon = '🎵';
                          else if (['jpg', 'jpeg', 'png', 'gif', 'webp'].includes(ext)) icon = '🖼';
                          else if (['zip', 'rar', 'tar', 'gz'].includes(ext)) icon = '📦';
                          
                          let sizeStr = '';
                          if (df.file_size < 1024) sizeStr = df.file_size + ' B';
                          else if (df.file_size < 1024*1024) sizeStr = (df.file_size/1024).toFixed(1) + ' KB';
                          else if (df.file_size < 1024*1024*1024) sizeStr = (df.file_size/1024/1024).toFixed(1) + ' MB';
                          else sizeStr = (df.file_size/1024/1024/1024).toFixed(2) + ' GB';

                          return (
                            <tr key={i} className="hover:bg-gray-800/50 transition-colors group">
                              <td className="py-3 px-4 text-center text-xl">{icon}</td>
                              <td className="py-3 px-4 font-medium text-gray-200 max-w-[200px] truncate" title={df.file_name}>{df.file_name}</td>
                              <td className="py-3 px-4 text-gray-400">{sizeStr}</td>
                              <td className="py-3 px-4 text-gray-500 text-xs">{df.downloaded_at}</td>
                              <td className="py-3 px-4 text-right opacity-50 group-hover:opacity-100 transition-opacity">
                                <div className="flex justify-end space-x-1">
                                  <button onClick={() => OpenFile(df.local_path)} className="px-3 py-1.5 text-xs font-bold text-gray-300 hover:text-white bg-gray-700 hover:bg-gray-600 rounded-md transition-colors flex items-center space-x-1" title={t('Ochish') || 'Open'}>
                                    <span>📂</span>
                                    <span>{t('Open')}</span>
                                  </button>
                                </div>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  )}
                </>
              )}

            </div>
          </div>
        </div>
      )}

      {/* Download Modal */}
      {showDownloadModal && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 animate-fade-in backdrop-blur-sm p-4">
          <div className="bg-gray-800 rounded-2xl border border-gray-700 shadow-2xl w-full max-w-md p-6 flex flex-col space-y-4 animate-fade-in-up">
            <div className="flex justify-between items-center mb-2">
              <h3 className="text-white font-bold text-lg">{t('Download via Link')}</h3>
              <button onClick={() => setShowDownloadModal(false)} className="text-gray-500 hover:text-white transition-colors">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>
            
            <p className="text-gray-400 text-sm">
              {t('Paste meshweb link')}
            </p>
            
            <input 
              type="text" 
              placeholder="meshweb://file/Qm..." 
              value={downloadLink}
              onChange={e => setDownloadLink(e.target.value)}
              className="w-full bg-gray-900 border border-gray-700 text-white px-4 py-3 rounded-xl outline-none focus:border-primary transition-colors font-mono text-sm"
              autoFocus
            />
            
            <div className="flex space-x-3 pt-2">
              <button 
                onClick={() => {
                  setShowDownloadModal(false);
                  handleDownload();
                }}
                disabled={loading || !downloadLink}
                className="flex-1 py-3 bg-primary hover:bg-primary/90 text-white rounded-xl font-bold transition-transform transform hover:scale-[1.02] active:scale-95 disabled:opacity-50 disabled:transform-none"
              >
                Yuklab olish
              </button>
            </div>
          </div>
        </div>
      )}

      {showRentModal && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 animate-fade-in backdrop-blur-sm p-4">
          <div className="bg-gray-800 rounded-3xl p-8 border border-gray-700 w-full max-w-sm shadow-2xl flex flex-col items-center">
            <h3 className="text-2xl font-bold mb-8 text-white">{t('Qancha vaqt kerak?')}</h3>

            {rentStep === 'form' && (
              <div className="space-y-6 w-full">
                <div className="grid grid-cols-3 gap-3">
                  {[1, 4, 8].map(h => (
                    <button 
                      key={h} 
                      onClick={() => setRentDuration(h)} 
                      className={`py-3 rounded-2xl text-lg font-bold transition-all border-2 ${rentDuration === h ? 'bg-primary/20 border-primary text-primary' : 'bg-gray-900 border-gray-700 text-gray-400 hover:bg-gray-700 hover:text-white'}`}
                    >
                      {h} {h === 1 ? t('soat_1') : t('soat')}
                    </button>
                  ))}
                </div>

                <div className="text-center py-4">
                  <span className="text-gray-400">{t('Narx')}: </span>
                  <span className="text-xl font-bold text-success">~{rentCost.toFixed(2)} MWC</span>
                </div>

                <div className="flex space-x-3 w-full">
                  <button onClick={() => setShowRentModal(false)} className="flex-1 py-4 bg-gray-700 hover:bg-gray-600 rounded-2xl text-white font-medium transition-all">Orqaga</button>
                  <button onClick={handleStartRent} className="flex-1 py-4 bg-success hover:bg-success/90 text-white rounded-2xl font-bold transition-all shadow-lg shadow-success/20">{t('Boshlash')}</button>
                </div>
              </div>
            )}

            {rentStep === 'finding' && (
              <div className="py-8 flex flex-col items-center space-y-6 w-full">
                <div className="w-16 h-16 border-4 border-primary border-t-transparent rounded-full animate-spin"></div>
                <span className="text-gray-300 font-medium animate-pulse">Node qidirilmoqda...</span>
              </div>
            )}

            {rentStep === 'active' && activeRentalStats && (
              <div className="space-y-6 w-full">
                <div className="flex flex-col items-center p-6 bg-gray-900 rounded-2xl border border-success/30">
                  <div className="w-16 h-16 bg-success/20 rounded-full flex items-center justify-center mb-4">
                    <span className="text-3xl">🟢</span>
                  </div>
                  <span className="text-success font-bold text-xl">{t('Connected')}</span>
                </div>

                <div className="text-center space-y-2">
                  <div className="text-gray-400">Qolgan vaqt: <span className="text-white font-mono font-bold">{(activeRentalStats.job.duration_hours - activeRentalStats.elapsed).toFixed(2)} hrs</span></div>
                  <div className="text-gray-400">Sarflangan: <span className="text-danger font-bold">-{activeRentalStats.job.spent_so_far.toFixed(4)} MWC</span></div>
                </div>

                <button onClick={handleStopRent} className="w-full py-4 bg-danger hover:bg-danger/90 text-white rounded-2xl font-bold transition-all shadow-lg shadow-danger/20">
                  {t("To'xtatish")}
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Bottom Bar */}
      <footer className="grid grid-cols-3 gap-4 pb-2">
        <label className={`cursor-pointer rounded-2xl flex flex-col items-center justify-center py-4 transition-all border-2 ${offerResources ? 'bg-success/20 border-success text-success shadow-[0_0_15px_rgba(74,222,128,0.2)]' : 'bg-gray-800 border-gray-700 text-gray-400 hover:bg-gray-700'}`}>
          <input type="checkbox" className="hidden" checked={offerResources} onChange={(e) => handleToggleOffer(e.target.checked)} />
          <svg className="w-8 h-8 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M13 10V3L4 14h7v7l9-11h-7z" /></svg>
          <span className="font-bold text-sm text-center">{t('Resurs taklif qilish')}</span>
        </label>

        <button onClick={() => setActiveTab('storage')} className={`rounded-2xl flex flex-col items-center justify-center py-4 transition-all border-2 ${activeTab === 'storage' ? 'bg-primary/20 border-primary text-primary' : 'bg-gray-800 border-gray-700 text-gray-400 hover:bg-gray-700 hover:text-white'}`}>
          <svg className="w-8 h-8 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" /></svg>
          <span className="font-bold text-sm">{t('Fayllarim')}</span>
        </button>

        <button onClick={() => setShowSettings(true)} className="bg-gray-800 border-2 border-gray-700 text-gray-400 hover:bg-gray-700 hover:text-white rounded-2xl flex flex-col items-center justify-center py-4 transition-all">
          <svg className="w-8 h-8 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /></svg>
          <span className="font-bold text-sm">{t('Sozlamalar')}</span>
        </button>
      </footer>

      {showSettings && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 animate-fade-in backdrop-blur-sm">
          <div className="bg-gray-800 rounded-2xl p-6 border border-gray-700 w-96 shadow-2xl flex flex-col max-h-[90vh] overflow-y-auto custom-scrollbar">
            <h3 className="text-xl font-bold mb-6 text-white text-center">{t('Sozlamalar')}</h3>
            
            <div className="mb-6 space-y-4">
              <h4 className="text-gray-400 text-sm font-bold uppercase tracking-wider">{t('Identity')}</h4>
              
              <div className="bg-gray-900 p-3 rounded-xl border border-gray-700 flex flex-col space-y-1">
                <span className="text-xs text-gray-500">{t('Public Key')}</span>
                <div className="flex justify-between items-center">
                  <span className="text-sm font-mono text-primary truncate mr-2">{myPublicKey}</span>
                  <button onClick={() => navigator.clipboard.writeText(myPublicKey)} className="text-gray-400 hover:text-white shrink-0">
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" /></svg>
                  </button>
                </div>
              </div>

              <button 
                onClick={async () => {
                  const res = await ExportIdentity();
                  if (res.success) alert(t('Your Seed Phrase') + ':\n\n' + res.seedPhrase);
                }}
                className="w-full text-left px-4 py-3 rounded-xl border border-gray-700 text-gray-300 hover:bg-gray-700 transition-all flex justify-between items-center"
              >
                <span>{t('Show Seed Phrase')}</span>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" /></svg>
              </button>

              <button 
                onClick={async () => {
                  const res = await ExportIdentity();
                  if (res.success) {
                    const blob = new Blob([JSON.stringify(res, null, 2)], { type: 'application/json' });
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = 'identity.json';
                    a.click();
                  }
                }}
                className="w-full text-left px-4 py-3 rounded-xl border border-gray-700 text-gray-300 hover:bg-gray-700 transition-all flex justify-between items-center"
              >
                <span>{t('Export Identity')}</span>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" /></svg>
              </button>

              <button 
                onClick={handleRegisterAssoc}
                className="w-full text-left px-4 py-3 rounded-xl border border-gray-700 text-gray-300 hover:bg-gray-700 transition-all flex justify-between items-center"
              >
                <span>{t('Register .meshweb')}</span>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 4v16m8-8H4" /></svg>
              </button>

              <button 
                onClick={() => { setShowSettings(false); setShowLogs(true); }}
                className="w-full text-left px-4 py-3 rounded-xl border border-gray-700 text-gray-300 hover:bg-gray-700 transition-all flex justify-between items-center"
              >
                <span>{t("Loglarni ko'rish")}</span>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
              </button>
            </div>

            <div className="mb-6 space-y-4">
              <h4 className="text-gray-400 text-sm font-bold uppercase tracking-wider">{t('Language')}</h4>
              <div className="space-y-2">
                <button onClick={() => changeLang('uz')} className={`w-full text-left px-4 py-2 rounded-xl border transition-all ${lang==='uz' ? 'border-primary bg-primary/10 text-primary' : 'border-gray-700 text-gray-300 hover:bg-gray-700'}`}>🇺🇿 O'zbek</button>
                <button onClick={() => changeLang('ru')} className={`w-full text-left px-4 py-2 rounded-xl border transition-all ${lang==='ru' ? 'border-primary bg-primary/10 text-primary' : 'border-gray-700 text-gray-300 hover:bg-gray-700'}`}>🇷🇺 Русский</button>
                <button onClick={() => changeLang('en')} className={`w-full text-left px-4 py-2 rounded-xl border transition-all ${lang==='en' ? 'border-primary bg-primary/10 text-primary' : 'border-gray-700 text-gray-300 hover:bg-gray-700'}`}>🇬🇧 English</button>
              </div>
            </div>

            <button onClick={() => setShowSettings(false)} className="w-full bg-gray-700 hover:bg-gray-600 py-3 rounded-xl text-white font-medium transition-all mt-auto">
              {t('Close')}
            </button>
          </div>
        </div>
      )}

      {showLogs && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 animate-fade-in backdrop-blur-sm p-4 md:p-6">
          <div className="bg-[#0D1117] rounded-2xl border border-gray-700 w-full max-w-3xl h-[80vh] flex flex-col shadow-2xl">
            <div className="flex justify-between items-center p-4 border-b border-gray-800">
              <h3 className="text-gray-300 font-bold tracking-wider">{t('Activity Log')}</h3>
              <button onClick={() => setShowLogs(false)} className="text-gray-500 hover:text-white transition-colors">
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>
            <div className="flex-grow overflow-y-auto p-4 space-y-2 font-mono text-sm custom-scrollbar">
              {logs.length === 0 ? (
                <div className="text-gray-600 italic">{t('No activity yet')}</div>
              ) : (
                logs.map((log, i) => (
                  <div key={i} className="animate-fade-in">
                    <span className="text-gray-500 mr-2">[{new Date().toLocaleTimeString()}]</span>
                    <span className={`${log.includes('Xato') || log.includes('Error') ? 'text-danger' : log.includes('✅') ? 'text-success' : 'text-gray-300'}`}>
                      {log}
                    </span>
                  </div>
                ))
              )}
              <div ref={logsEndRef} />
            </div>
          </div>
        </div>
      )}

      {/* Compute Coming Soon Modal */}
      {showComputeModal && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 animate-fade-in backdrop-blur-sm p-4">
          <div className="bg-gray-800 rounded-2xl border border-gray-700 shadow-2xl w-full max-w-sm p-8 flex flex-col items-center space-y-6 text-center animate-fade-in-up">
            <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center">
              <span className="text-4xl">⚡</span>
            </div>
            <div>
              <h3 className="text-white font-bold text-xl mb-2">Compute Market</h3>
              <p className="text-gray-400 text-sm leading-relaxed whitespace-pre-line">
                {t('Compute coming soon text')}
              </p>
            </div>
            <button
              onClick={() => setShowComputeModal(false)}
              className="w-full py-3 bg-primary hover:bg-primary/90 text-white rounded-xl font-bold transition-all transform hover:scale-[1.02] active:scale-95"
            >
              OK
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
