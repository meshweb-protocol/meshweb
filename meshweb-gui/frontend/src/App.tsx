import { useState, useEffect, useRef } from 'react';
import { StartNewNetwork, ConnectToNetwork, GetDashboardStats, ToggleOfferResources, LoadIdentity, GenerateIdentity, RestoreIdentity, GetPublicKey, ExportIdentity, SelectFile, SelectFolder, UploadFile, UploadFolder, DownloadFile, GenerateShareLink, GenerateMeshwebFile, GetMyFiles, DeleteFile, RegisterFileAssociation, GetStartupFile, FindAvailableNodes, StartRental, StopRental, GetRentalStatus, GetDownloadedFiles, OpenFile } from '../wailsjs/go/main/App';
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
  const [activeTab, setActiveTab] = useState<'dashboard' | 'files' | 'settings'>('dashboard');
  const [storageSubTab, setStorageSubTab] = useState<'uploads' | 'downloads'>('uploads');
  const [myFiles, setMyFiles] = useState<any[]>([]);
  const [downloadedFiles, setDownloadedFiles] = useState<any[]>([]);
  const [currentFolder, setCurrentFolder] = useState<any>(null);
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

  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' | 'info' } | null>(null);

  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'success') => {
    setToast({ message, type });
    setTimeout(() => setToast(null), 2500);
  };

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
        setActiveTab('files');
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

  const findFolderRecursive = (items: any[], id: string): any => {
    for (const item of items) {
      if (item.file_id === id) return item;
      if (item.files) {
        const found = findFolderRecursive(item.files, id);
        if (found) return found;
      }
    }
    return null;
  };

  const loadMyFiles = async () => {
    const files = await GetMyFiles();
    setMyFiles(files || []);
    const downloaded = await GetDownloadedFiles();
    setDownloadedFiles(downloaded || []);
    
    setCurrentFolder((prev: any) => {
      if (!prev) return null;
      const updated = findFolderRecursive(files || [], prev.file_id);
      return updated || null;
    });
  };

  useEffect(() => {
    if (activeTab === 'files') {
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
    
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      const file: any = e.dataTransfer.files[0];
      const path = file.path || file.name;
      if (path && path.includes('\\')) {
        setLoading(true);
        // Check if it's a folder (no extension and size 0 often indicates folder)
        if (file.type === '' && file.size === 0) {
          const res = await UploadFolder(path);
          if (res.success) loadMyFiles();
          else showToast("Upload failed: " + res.error, "error");
        } else {
          const res = await UploadFile(path);
          if (res.success) loadMyFiles();
          else showToast("Upload failed: " + res.error, "error");
        }
        setLoading(false);
      } else {
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
      showToast("Upload failed: " + res.error, "error");
    }
    setLoading(false);
  };

  const handleUploadFolder = async () => {
    const path = await SelectFolder();
    if (!path) return;
    setLoading(true);
    const res = await UploadFolder(path);
    if (res.success) {
      loadMyFiles();
    } else {
      showToast("Upload failed: " + res.error, "error");
    }
    setLoading(false);
  };

  const handleDownload = async () => {
    if (!downloadLink) return;
    setLoading(true);
    setDownloadProgress(0);
    const res = await DownloadFile(downloadLink);
    if (res.success) {
      showToast(t('Download') + " ✅ " + res.path);
      loadMyFiles();
    } else {
      showToast("Error: " + res.error, "error");
    }
    setDownloadProgress(-1);
    setLoading(false);
    setDownloadLink('');
  };

  const handleShareLink = async (fileId: string) => {
    const link = await GenerateShareLink(fileId);
    if (link) {
      navigator.clipboard.writeText(link);
      showToast(t('Link copied!'));
    } else {
      showToast('Error generating link', 'error');
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
    const res = await RegisterFileAssociation();
    if (res.success) {
      showToast("Windows File Association Registered ✅");
    }
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
          setActiveRentalJobId(startRes.job_id);
          setRentStep('active');
        } else {
          showToast("Failed to rent: " + startRes.error, "error");
          setRentStep('form');
        }
      }, 1500);
    } else {
      showToast("No suitable nodes available.", "error");
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
      <div className="h-screen bg-[var(--bg-primary)] flex flex-col items-center justify-center text-[var(--text-primary)] font-sans px-5">
        <div className="w-full max-w-[360px] flex flex-col items-center space-y-8 animate-fade-in">
          <div className="flex flex-col items-center space-y-3">
            <img src="/logo.png" width="64" height="64" style={{borderRadius: '8px'}}/>
            <h1 className="text-[24px] font-bold">Meshweb</h1>
            <p className="text-[13px] text-[var(--text-secondary)]">{t('Welcome to Meshweb')}</p>
          </div>

          <div className="w-full flex flex-col space-y-4">
            {error && <div className="text-danger text-sm text-center bg-danger/10 p-2 rounded">{error}</div>}
            
            {onboardingView === 'main' && (
              <div className="space-y-4">
                <button
                  onClick={handleCreateIdentity}
                  disabled={loading}
                  className="w-full py-2 bg-[var(--accent)] text-white rounded-[6px] text-[13px] font-medium transition-opacity hover:opacity-90 disabled:opacity-50 border-none mb-3"
                >
                  {t('Create New Account')}
                </button>
                <div className="relative flex items-center py-2 mb-3">
                  <div className="flex-grow border-t border-[var(--border)]"></div>
                  <span className="flex-shrink-0 mx-4 text-[var(--text-secondary)] text-[11px]">{t('YOKI')}</span>
                  <div className="flex-grow border-t border-[var(--border)]"></div>
                </div>
                <button
                  onClick={() => setOnboardingView('restore')}
                  className="w-full py-2 bg-transparent border border-[var(--border)] text-[var(--text-secondary)] hover:text-white hover:border-[var(--text-muted)] rounded-[6px] text-[13px] transition-colors"
                >
                  {t('Restore Account')}
                </button>
              </div>
            )}

            {onboardingView === 'create' && (
              <div className="space-y-6">
                <h3 className="text-[13px] font-semibold text-white">{t('Your Seed Phrase')}</h3>
                <div className="bg-[var(--bg-secondary)] border border-[var(--border)] p-3 rounded-[6px] font-mono text-[12px] text-[var(--accent)] select-all break-words">
                  {seedPhrase}
                </div>
                <label className="flex items-center space-x-2 cursor-pointer mt-2 mb-2">
                  <input type="checkbox" checked={savedSeed} onChange={(e) => setSavedSeed(e.target.checked)} className="rounded text-[var(--accent)] focus:ring-[var(--accent)] bg-[var(--bg-secondary)] border-[var(--border)]" />
                  <span className="text-[12px] text-[var(--text-secondary)]">{t('I have saved my seed phrase')}</span>
                </label>
                <button
                  onClick={handleFinishCreate}
                  disabled={!savedSeed}
                  className="w-full py-2 bg-[var(--accent)] text-white rounded-[6px] text-[13px] font-medium transition-opacity hover:opacity-90 disabled:opacity-50 border-none"
                >
                  Davom etish
                </button>
              </div>
            )}

            {onboardingView === 'restore' && (
              <div className="space-y-4">
                <h3 className="text-[13px] font-semibold text-white">{t('Restore Account')}</h3>
                <textarea
                  value={inputSeed}
                  onChange={(e) => setInputSeed(e.target.value)}
                  placeholder="word1 word2 word3..."
                  className="w-full h-24 bg-[var(--bg-secondary)] border border-[var(--border)] text-white p-3 rounded-[6px] text-[12px] outline-none focus:border-[var(--accent)] custom-scrollbar resize-none"
                />
                <button
                  onClick={handleRestoreIdentity}
                  disabled={loading || !inputSeed}
                  className="w-full py-2 bg-[var(--accent)] text-white rounded-[6px] text-[13px] font-medium transition-opacity hover:opacity-90 disabled:opacity-50 border-none"
                >
                  Davom etish
                </button>
                <button onClick={() => setOnboardingView('main')} className="w-full text-[12px] text-[var(--text-secondary)] hover:text-white transition-colors mt-2">Orqaga</button>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  if (!connected) {
    return (
      <div className="h-screen bg-[var(--bg-primary)] flex flex-col items-center justify-center text-[var(--text-primary)] font-sans px-5">
        <div className="w-full max-w-[360px] flex flex-col items-center space-y-8 animate-fade-in">
          {/* Logo Section */}
          <div className="flex flex-col items-center space-y-3">
            <img src="/logo.png" width="64" height="64" style={{borderRadius: '8px'}}/>
            <h1 className="text-[24px] font-bold">Meshweb</h1>
            <p className="text-[13px] text-[var(--text-secondary)] text-center">Decentralized Compute Network</p>
          </div>

          <div className="w-full flex flex-col items-center space-y-4">
            {error && <div className="text-[var(--danger)] text-[12px] text-center w-full">{error}</div>}
            
            <button
              onClick={handleStartNetwork}
              disabled={loading}
              className="w-full py-2 bg-[var(--accent)] text-white rounded-[6px] text-[13px] font-medium transition-opacity hover:opacity-90 disabled:opacity-50 border-none"
            >
              {loading ? t('Kutish...') : t('Join Meshweb')}
            </button>
            <p className="text-[var(--text-secondary)] text-[11px] text-center mt-2">
              Decentralized resources powered by community.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="h-screen bg-[var(--bg-primary)] text-[var(--text-primary)] font-sans flex flex-col overflow-hidden">
      {/* Top Bar */}
      <header className="flex justify-between items-center bg-[var(--bg-primary)] border-b border-[var(--border)] h-[52px] px-5">
        <div className="flex items-center space-x-2 cursor-pointer" onClick={() => setActiveTab('dashboard')}>
          <img src="/logo.png" width="24" height="24" style={{borderRadius: '4px'}}/>
          <h2 className="text-[15px] font-semibold text-white">Meshweb</h2>
        </div>
        <div className="flex items-center space-x-6">
          <div className="flex items-center space-x-2">
            <span className="w-1.5 h-1.5 rounded-full bg-[var(--success)]"></span>
            <span className="text-[12px] text-[var(--text-secondary)]">{t('Connected')}</span>
          </div>
          <div className="flex items-center px-2 py-1 bg-[var(--bg-tertiary)] rounded-[6px] border border-[var(--border)]" title={myPublicKey}>
            <span className="text-[12px] font-mono text-white">MW-{myPublicKey.substring(myPublicKey.length - 4)}</span>
          </div>
        </div>
      </header>

      {activeTab === 'dashboard' && (
        <div className="flex-grow flex flex-col px-5 py-6 overflow-y-auto custom-scrollbar animate-fade-in">
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 w-full max-w-2xl mx-auto mt-8">
            {/* Balance Card */}
            <div className="bg-[var(--bg-secondary)] border border-[var(--border)] rounded-lg p-5 flex flex-col">
              <span className="text-[10px] text-[var(--text-secondary)] tracking-[1px] mb-2 uppercase">{t('BALANCE')}</span>
              <span className="text-[28px] font-bold text-white mb-1">{stats.balance.toFixed(2)} MWC</span>
              <span className="text-[12px] text-[var(--text-secondary)]">{t('Today')} +{stats.todayIncome.toFixed(2)} MWC</span>
            </div>

            {/* Network Card */}
            <div className="bg-[var(--bg-secondary)] border border-[var(--border)] rounded-lg p-5 flex flex-col">
              <span className="text-[10px] text-[var(--text-secondary)] tracking-[1px] mb-2 uppercase">{t('NETWORK')}</span>
              <span className="text-[28px] font-bold text-white mb-1">{stats.connectedPeers} nodes</span>
              <span className="text-[12px] text-[var(--text-secondary)]">~45ms</span>
            </div>
            
            {/* Offer Resources Toggle */}
            <div className="md:col-span-2 flex items-center justify-between py-4 border-t border-[var(--border)] mt-4">
              <span className="text-[13px] text-white">{t('Offer Resources')}</span>
              <label className="relative inline-flex items-center cursor-pointer">
                <input type="checkbox" className="sr-only peer" checked={offerResources} onChange={(e) => handleToggleOffer(e.target.checked)} />
                <div className={`w-9 h-5 rounded-full peer peer-checked:bg-[var(--success)] bg-[var(--border)] transition-colors after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-[#555555] after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full peer-checked:after:bg-white`}></div>
              </label>
            </div>
            
            {/* Compute Market Link */}
            <div className="md:col-span-2 flex items-center justify-between py-4 border-t border-[var(--border)] cursor-pointer group" onClick={() => setShowComputeModal(true)}>
              <span className="text-[13px] text-white group-hover:text-[var(--accent)] transition-colors">{t('Compute Market')}</span>
              <span className="text-[11px] text-[var(--accent)] bg-[var(--accent)]/10 px-2 py-0.5 rounded-[4px] uppercase tracking-wide">Coming Soon</span>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'files' && (
        <div 
          className="flex-grow flex flex-col px-5 py-6 animate-fade-in"
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
        >
          <div className={`w-full bg-[var(--bg-primary)] flex flex-col h-full overflow-hidden transition-colors ${isDragOver ? 'border border-[var(--success)] bg-[var(--bg-secondary)]' : ''}`}>
            
            {/* Top Bar with Tabs */}
            <div className="flex justify-between items-center p-3 border-b border-[var(--border)] bg-[var(--bg-secondary)]">
              <div className="flex space-x-6 items-center">
                <h2 className="text-[15px] font-semibold text-white px-2">My Files</h2>
                
                <button 
                  onClick={() => setStorageSubTab('uploads')}
                  className={`text-[13px] font-medium transition-colors pb-1 ${storageSubTab === 'uploads' ? 'text-white border-b-2 border-[var(--accent)]' : 'text-[var(--text-secondary)] hover:text-white'}`}
                >
                  {t('My Uploads')}
                </button>
                <button 
                  onClick={() => setStorageSubTab('downloads')}
                  className={`text-[13px] font-medium transition-colors pb-1 ${storageSubTab === 'downloads' ? 'text-white border-b-2 border-[var(--accent)]' : 'text-[var(--text-secondary)] hover:text-white'}`}
                >
                  {t('Downloaded')}
                </button>
              </div>

              <div className="flex space-x-2">
                <button 
                  onClick={() => setShowDownloadModal(true)}
                  className="px-3 py-1.5 bg-[var(--bg-tertiary)] hover:bg-[var(--bg-hover)] text-white border border-[var(--border)] rounded-[6px] text-[12px] transition-colors"
                >
                  {t('Link orqali yuklab olish') || 'Download via Link'}
                </button>
                <button 
                  onClick={handleUploadFolder}
                  disabled={loading}
                  className="px-3 py-1.5 bg-[var(--bg-tertiary)] hover:bg-[var(--bg-hover)] text-white border border-[var(--border)] rounded-[6px] text-[12px] transition-colors disabled:opacity-50"
                >
                  {t('Upload Folder')}
                </button>
                <button 
                  onClick={handleUpload}
                  disabled={loading}
                  className="px-3 py-1.5 bg-[var(--accent)] hover:opacity-90 text-white rounded-[6px] text-[12px] transition-opacity disabled:opacity-50 border-none"
                >
                  + {t('Fayl yuklash') || 'Upload'}
                </button>
              </div>
            </div>

            {/* Drag Overlay Bar if files exist */}
            {isDragOver && (myFiles.length > 0 || storageSubTab === 'downloads') && (
              <div className="bg-[var(--success)] text-black text-center py-2 text-[12px] font-medium">
                {t('Yangi fayl yuklash uchun tashlang') || 'Drop files here to upload'}
              </div>
            )}

            {/* Content */}
            <div className="flex-grow overflow-y-auto custom-scrollbar">
              
              {/* UPLOADS TAB */}
              {storageSubTab === 'uploads' && (
                <>
                  {myFiles.length === 0 && downloadProgress < 0 ? (
                    <div className="flex flex-col items-center justify-center h-full space-y-4 text-[var(--text-secondary)] border border-dashed border-[var(--border)] m-6 rounded-lg p-12 bg-[var(--bg-secondary)]">
                      <p className="text-[15px] font-medium text-white">{t('Drop files here')}</p>
                      <p className="text-[12px] text-[var(--text-secondary)]">{t('or')}</p>
                      <button 
                        onClick={handleUpload}
                        disabled={loading}
                        className="mt-4 py-2 px-6 bg-[var(--accent)] hover:opacity-90 text-white rounded-[6px] font-medium transition-opacity disabled:opacity-50 border-none"
                      >
                        + {t('Fayl yuklash') || 'Upload File'}
                      </button>
                    </div>
                  ) : (
                    <div className="bg-[var(--bg-primary)]">
                      {currentFolder && (
                        <div className="p-3 border-b border-[var(--border)] flex items-center space-x-3 bg-[var(--bg-secondary)]">
                          <span className="font-semibold text-[13px] text-white ml-2">📁 {currentFolder.file_name}</span>
                        </div>
                      )}
                      <table className="w-full text-left border-collapse">
                        <thead className="bg-[var(--bg-secondary)] sticky top-0 z-10 border-b border-[var(--border)] text-[11px] uppercase text-[var(--text-secondary)] tracking-wider">
                        <tr>
                          <th className="py-2 px-4 w-10 text-center"></th>
                          <th className="py-2 px-4 font-semibold">{t('Name')}</th>
                          <th className="py-2 px-4 font-semibold w-24">{t('Size')}</th>
                          <th className="py-2 px-4 font-semibold w-32">{t('Status')}</th>
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
                        
                        {(currentFolder ? currentFolder.files : myFiles).map((f: any, i: number) => {
                          let icon = '📄';
                          if (f.type === 'folder') icon = '📁';
                          else if (f.file_name.endsWith('.png') || f.file_name.endsWith('.jpg')) icon = '🖼️';
                          else if (f.file_name.endsWith('.mp4')) icon = '🎬';
                          else if (f.file_name.endsWith('.zip')) icon = '📦';

                          let sizeStr = '';
                          if (f.file_size < 1024) sizeStr = f.file_size + ' B';
                          else if (f.file_size < 1024*1024) sizeStr = (f.file_size/1024).toFixed(1) + ' KB';
                          else if (f.file_size < 1024*1024*1024) sizeStr = (f.file_size/1024/1024).toFixed(2) + ' MB';
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
                            <tr key={i} className="border-b border-[var(--bg-tertiary)] hover:bg-[var(--bg-secondary)] transition-colors cursor-pointer" onDoubleClick={() => {
                              if (f.type === 'folder') setCurrentFolder(f);
                              else handleShareLink(f.file_id);
                            }}>
                              <td className="py-3 px-4 text-center text-xl">{icon}</td>
                              <td className="py-3 px-4 font-medium text-white max-w-[200px] truncate" title={f.file_name}>{f.file_name}</td>
                              <td className="py-3 px-4 text-[var(--text-secondary)]">{sizeStr}</td>
                              <td className="py-3 px-4">{stateBadge}</td>
                              <td className="py-3 px-4 text-right">
                                <div className="flex justify-end space-x-2 items-center">
                                  {f.local_path && (
                                    <button onClick={() => OpenFile(f.local_path)} className="p-1.5 text-[16px] text-[var(--text-secondary)] hover:text-white transition-colors" title={t('Ochish') || 'Open'}>📂</button>
                                  )}
                                  <button onClick={() => handleShareLink(f.file_id)} className="p-1.5 text-[16px] text-[var(--text-secondary)] hover:text-white transition-colors" title={t('Ulashish')}>🔗</button>
                                  <button onClick={() => handleDeleteFile(f.file_id)} className="p-1.5 text-[16px] text-[var(--text-secondary)] hover:text-white transition-colors" title="Delete">🗑</button>
                                </div>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                      </table>
                    </div>
                  )}
                </>
              )}

              {/* DOWNLOADS TAB */}
              {storageSubTab === 'downloads' && (
                <>
                  {downloadedFiles.length === 0 ? (
                    <div className="flex flex-col items-center justify-center h-full text-[13px] text-[var(--text-muted)]">
                      {t('No files yet')}
                    </div>
                  ) : (
                    <div className="bg-[var(--bg-primary)]">
                      <table className="w-full text-left border-collapse">
                        <thead className="bg-[var(--bg-secondary)] sticky top-0 z-10 border-b border-[var(--border)] text-[11px] uppercase text-[var(--text-secondary)] tracking-wider">
                        <tr>
                          <th className="py-2 px-4 w-10 text-center"></th>
                          <th className="py-2 px-4 font-semibold">{t('Name')}</th>
                          <th className="py-2 px-4 font-semibold w-24">{t('Size')}</th>
                          <th className="py-2 px-4 font-semibold w-40">{t('Date')}</th>
                          <th className="py-2 px-4 font-semibold w-32 text-right">{t('Actions')}</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-[var(--border)] text-[13px]">
                        {downloadedFiles.map((df, i) => {
                          const ext = df.file_name.split('.').pop()?.toLowerCase();
                          let icon = '📄';
                          if (df.type === 'folder') icon = '📁';
                          else if (['mp4', 'avi', 'mkv', 'mov'].includes(ext)) icon = '📹';
                          else if (['mp3', 'wav', 'ogg'].includes(ext)) icon = '🎵';
                          else if (['jpg', 'jpeg', 'png', 'gif', 'webp'].includes(ext)) icon = '🖼';
                          else if (['zip', 'rar', 'tar', 'gz'].includes(ext)) icon = '📦';
                          
                          let sizeStr = '';
                          if (df.file_size < 1024) sizeStr = df.file_size + ' B';
                          else if (df.file_size < 1024*1024) sizeStr = (df.file_size/1024).toFixed(1) + ' KB';
                          else if (df.file_size < 1024*1024*1024) sizeStr = (df.file_size/1024/1024).toFixed(1) + ' MB';
                          else sizeStr = (df.file_size/1024/1024/1024).toFixed(2) + ' GB';

                          return (
                            <tr key={i} className="border-b border-[var(--bg-tertiary)] hover:bg-[var(--bg-secondary)] transition-colors cursor-pointer" onDoubleClick={() => OpenFile(df.local_path)}>
                              <td className="py-3 px-4 text-center text-xl">{icon}</td>
                              <td className="py-3 px-4 font-medium text-white max-w-[200px] truncate" title={df.file_name}>{df.file_name}</td>
                              <td className="py-3 px-4 text-[var(--text-secondary)]">{sizeStr}</td>
                              <td className="py-3 px-4 text-[var(--text-muted)] text-[11px]">{df.downloaded_at}</td>
                              <td className="py-3 px-4 text-right">
                                <button onClick={() => OpenFile(df.local_path)} className="p-1.5 text-[16px] text-[var(--text-secondary)] hover:text-white transition-colors" title={t('Ochish') || 'Open'}>📂</button>
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                      </table>
                    </div>
                  )}
                </>
              )}

            </div>
          </div>
        </div>
      )}

      {/* Download Modal */}
      {/* Download Modal */}
      {showDownloadModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 animate-fade-in" onClick={(e) => { if (e.target === e.currentTarget) setShowDownloadModal(false) }}>
          <div className="bg-[var(--bg-secondary)] border border-[var(--border)] rounded-[8px] p-6 w-[400px] flex flex-col space-y-4">
            <div className="flex justify-between items-center mb-2">
              <h3 className="text-white font-semibold text-[15px]">{t('Download via Link')}</h3>
              <button onClick={() => setShowDownloadModal(false)} className="text-[var(--text-secondary)] hover:text-white transition-colors">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>
            
            <p className="text-[var(--text-secondary)] text-[13px]">
              {t('Paste meshweb link')}
            </p>
            
            <input 
              type="text" 
              placeholder="meshweb://file/Qm..." 
              value={downloadLink}
              onChange={e => setDownloadLink(e.target.value)}
              className="w-full bg-[var(--bg-primary)] border border-[var(--border)] text-white p-[10px_12px] rounded-[6px] outline-none focus:border-[var(--accent)] transition-colors font-mono text-[13px]"
              autoFocus
            />
            
            <div className="flex pt-2">
              <button 
                onClick={() => {
                  setShowDownloadModal(false);
                  handleDownload();
                }}
                disabled={loading || !downloadLink}
                className="w-full p-[10px] bg-[var(--accent)] text-white rounded-[6px] text-[13px] font-medium transition-opacity hover:opacity-90 disabled:opacity-50 border-none"
              >
                Yuklab olish
              </button>
            </div>
          </div>
        </div>
      )}

      {showRentModal && (
        <div className="fixed inset-0 bg-black/70 flex items-center justify-center z-50 animate-fade-in" onClick={(e) => { if (e.target === e.currentTarget) setShowRentModal(false) }}>
          <div className="bg-[var(--bg-secondary)] border border-[var(--border)] rounded-[8px] p-6 w-full max-w-sm flex flex-col items-center">
            <h3 className="text-[15px] font-semibold mb-6 text-white">{t('Qancha vaqt kerak?')}</h3>

            {rentStep === 'form' && (
              <div className="space-y-6 w-full">
                <div className="grid grid-cols-3 gap-3">
                  {[1, 4, 8].map(h => (
                    <button 
                      key={h} 
                      onClick={() => setRentDuration(h)} 
                      className={`py-2 rounded-[6px] text-[13px] font-medium transition-colors border ${rentDuration === h ? 'bg-[var(--accent)] border-[var(--accent)] text-white' : 'bg-[var(--bg-primary)] border-[var(--border)] text-[var(--text-secondary)] hover:border-[var(--text-muted)] hover:text-white'}`}
                    >
                      {h} {h === 1 ? t('soat_1') : t('soat')}
                    </button>
                  ))}
                </div>

                <div className="text-center py-2">
                  <span className="text-[13px] text-[var(--text-secondary)]">{t('Narx')}: </span>
                  <span className="text-[15px] font-bold text-[var(--success)]">~{rentCost.toFixed(2)} MWC</span>
                </div>

                <div className="flex space-x-3 w-full">
                  <button onClick={() => setShowRentModal(false)} className="flex-1 py-2.5 bg-[var(--bg-tertiary)] border border-[var(--border)] hover:border-[var(--text-muted)] rounded-[6px] text-white text-[13px] transition-colors">Orqaga</button>
                  <button onClick={handleStartRent} className="flex-1 py-2.5 bg-[var(--success)] hover:bg-[var(--success)]/90 text-[var(--bg-primary)] rounded-[6px] text-[13px] font-semibold transition-colors border-none">{t('Boshlash')}</button>
                </div>
              </div>
            )}

            {rentStep === 'finding' && (
              <div className="py-6 flex flex-col items-center space-y-4 w-full">
                <div className="w-8 h-8 border-2 border-[var(--accent)] border-t-transparent rounded-full animate-spin"></div>
                <span className="text-[12px] text-[var(--text-secondary)] animate-pulse">Node qidirilmoqda...</span>
              </div>
            )}

            {rentStep === 'active' && activeRentalStats && (
              <div className="space-y-6 w-full">
                <div className="flex flex-col items-center p-4 bg-[var(--bg-primary)] rounded-[6px] border border-[var(--success)]">
                  <div className="w-10 h-10 bg-[var(--success)]/20 rounded-full flex items-center justify-center mb-3">
                    <span className="text-xl">🟢</span>
                  </div>
                  <span className="text-[var(--success)] font-semibold text-[13px]">{t('Connected')}</span>
                </div>

                <div className="text-center space-y-2 text-[12px]">
                  <div className="text-[var(--text-secondary)]">Qolgan vaqt: <span className="text-white font-mono">{(activeRentalStats.job.duration_hours - activeRentalStats.elapsed).toFixed(2)} hrs</span></div>
                  <div className="text-[var(--text-secondary)]">Sarflangan: <span className="text-[var(--danger)]">-{activeRentalStats.job.spent_so_far.toFixed(4)} MWC</span></div>
                </div>

                <button onClick={handleStopRent} className="w-full py-2.5 bg-[var(--danger)] hover:opacity-90 text-white rounded-[6px] text-[13px] font-semibold transition-opacity border-none">
                  {t("To'xtatish")}
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Bottom Bar */}
      <footer className="h-[56px] bg-[var(--bg-primary)] border-t border-[var(--border)] grid grid-cols-3">
        <button onClick={() => setActiveTab('dashboard')} className={`flex items-center justify-center font-medium text-[13px] transition-colors ${activeTab === 'dashboard' ? 'text-white border-t-2 border-[var(--accent)]' : 'text-[var(--text-muted)] border-t-2 border-transparent hover:text-[var(--text-secondary)]'}`}>
          {t('Dashboard')}
        </button>

        <button onClick={() => { setActiveTab('files'); setCurrentFolder(null); }} className={`flex items-center justify-center font-medium text-[13px] transition-colors ${activeTab === 'files' ? 'text-white border-t-2 border-[var(--accent)]' : 'text-[var(--text-muted)] border-t-2 border-transparent hover:text-[var(--text-secondary)]'}`}>
          {t('Fayllarim') || 'My Files'}
        </button>

        <button onClick={() => setShowSettings(true)} className={`flex items-center justify-center font-medium text-[13px] transition-colors ${showSettings ? 'text-white border-t-2 border-[var(--accent)]' : 'text-[var(--text-muted)] border-t-2 border-transparent hover:text-[var(--text-secondary)]'}`}>
          {t('Sozlamalar') || 'Settings'}
        </button>
      </footer>

      {showSettings && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 animate-fade-in" onClick={(e) => { if (e.target === e.currentTarget) setShowSettings(false) }}>
          <div className="bg-[var(--bg-secondary)] border border-[var(--border)] rounded-lg p-6 w-[360px] flex flex-col">
            <h3 className="text-[15px] font-semibold text-white mb-6">{t('Sozlamalar') || 'Settings'}</h3>
            
            <div className="mb-6 space-y-3">
              <h4 className="text-[10px] text-[var(--text-secondary)] uppercase tracking-widest">{t('Identity')}</h4>
              
              <button 
                onClick={async () => {
                  const res = await ExportIdentity();
                  if (res.success) showToast(t('Your Seed Phrase') + ':\n\n' + res.seedPhrase, 'info');
                }}
                className="w-full text-left px-3 py-2 bg-[var(--bg-tertiary)] rounded-[6px] border border-[var(--border)] text-[var(--text-secondary)] hover:text-white hover:border-[var(--text-muted)] transition-colors flex justify-between items-center text-[13px]"
              >
                <span>{t('Show Seed Phrase')}</span>
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
                className="w-full text-left px-3 py-2 bg-[var(--bg-tertiary)] rounded-[6px] border border-[var(--border)] text-[var(--text-secondary)] hover:text-white hover:border-[var(--text-muted)] transition-colors flex justify-between items-center text-[13px]"
              >
                <span>{t('Export Identity')}</span>
              </button>

              <button 
                onClick={handleRegisterAssoc}
                className="w-full text-left px-3 py-2 bg-[var(--bg-tertiary)] rounded-[6px] border border-[var(--border)] text-[var(--text-secondary)] hover:text-white hover:border-[var(--text-muted)] transition-colors flex justify-between items-center text-[13px]"
              >
                <span>{t('Register .meshweb')}</span>
              </button>

              <button 
                onClick={() => { setShowSettings(false); setShowLogs(true); }}
                className="w-full text-left px-3 py-2 bg-[var(--bg-tertiary)] rounded-[6px] border border-[var(--border)] text-[var(--text-secondary)] hover:text-white hover:border-[var(--text-muted)] transition-colors flex justify-between items-center text-[13px]"
              >
                <span>{t("Loglarni ko'rish")}</span>
              </button>
            </div>

            <div className="w-full h-[1px] bg-[var(--border)] mb-6"></div>

            <div className="space-y-3">
              <h4 className="text-[10px] text-[var(--text-secondary)] uppercase tracking-widest">{t('Language')}</h4>
              <div className="flex flex-col space-y-2">
                <button onClick={() => changeLang('uz')} className={`w-full text-left px-3 py-2 rounded-[6px] border text-[13px] transition-colors ${lang==='uz' ? 'border-[var(--accent)] text-[var(--accent)] bg-[var(--bg-primary)]' : 'border-[var(--border)] text-[var(--text-secondary)] bg-[var(--bg-tertiary)] hover:border-[var(--text-muted)]'}`}>🇺🇿 O'zbek</button>
                <button onClick={() => changeLang('ru')} className={`w-full text-left px-3 py-2 rounded-[6px] border text-[13px] transition-colors ${lang==='ru' ? 'border-[var(--accent)] text-[var(--accent)] bg-[var(--bg-primary)]' : 'border-[var(--border)] text-[var(--text-secondary)] bg-[var(--bg-tertiary)] hover:border-[var(--text-muted)]'}`}>🇷🇺 Русский</button>
                <button onClick={() => changeLang('en')} className={`w-full text-left px-3 py-2 rounded-[6px] border text-[13px] transition-colors ${lang==='en' ? 'border-[var(--accent)] text-[var(--accent)] bg-[var(--bg-primary)]' : 'border-[var(--border)] text-[var(--text-secondary)] bg-[var(--bg-tertiary)] hover:border-[var(--text-muted)]'}`}>🇬🇧 English</button>
              </div>
            </div>
          </div>
        </div>
      )}

      {showLogs && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 animate-fade-in" onClick={(e) => { if (e.target === e.currentTarget) setShowLogs(false) }}>
          <div className="bg-[var(--bg-primary)] border border-[var(--border)] rounded-[8px] w-[90%] max-w-3xl h-[80vh] flex flex-col shadow-2xl">
            <div className="flex justify-between items-center p-4 border-b border-[var(--border)]">
              <h3 className="text-white text-[13px] font-semibold tracking-wider">{t('Activity Log')}</h3>
              <button onClick={() => setShowLogs(false)} className="text-[var(--text-secondary)] hover:text-white transition-colors">
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" /></svg>
              </button>
            </div>
            <div className="flex-grow overflow-y-auto p-4 space-y-2 font-mono text-[12px] custom-scrollbar">
              {logs.length === 0 ? (
                <div className="text-[var(--text-muted)] italic">{t('No activity yet')}</div>
              ) : (
                logs.map((log, i) => {
                  let logColor = 'text-[#CCCCCC]';
                  if (log.includes('Xato') || log.includes('Error') || log.includes('[Error]')) logColor = 'text-[var(--danger)]';
                  else if (log.includes('✅')) logColor = 'text-[var(--success)]';

                  // Extract prefix if exists
                  const prefixMatch = log.match(/^(\[.*?\])\s(.*)/);
                  let prefix = '';
                  let text = log;
                  if (prefixMatch) {
                    prefix = prefixMatch[1];
                    text = prefixMatch[2];
                  }

                  return (
                  <div key={i} className="animate-fade-in flex space-x-2">
                    <span className="text-[var(--text-muted)] shrink-0">[{new Date().toLocaleTimeString()}]</span>
                    <span className={logColor}>
                      {prefix && <span className="text-[var(--text-secondary)] mr-2">{prefix}</span>}
                      {text}
                    </span>
                  </div>
                  );
                })
              )}
              <div ref={logsEndRef} />
            </div>
          </div>
        </div>
      )}

      {/* Compute Coming Soon Modal */}
      {showComputeModal && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 animate-fade-in" onClick={(e) => { if (e.target === e.currentTarget) setShowComputeModal(false) }}>
          <div className="bg-[var(--bg-secondary)] border border-[var(--border)] rounded-lg p-8 w-[360px] flex flex-col items-center text-center">
            <span className="text-4xl mb-4">⚡</span>
            <h3 className="text-[15px] font-semibold text-white mb-2">Compute Market</h3>
            <p className="text-[13px] text-[var(--text-secondary)] mb-6">
              AI modellarni o'rgatish va yurgizish uchun markazlashmagan hisoblash quvvati tez kunda qo'shiladi.
            </p>
            <button onClick={() => setShowComputeModal(false)} className="w-full py-2 bg-[var(--bg-tertiary)] border border-[var(--border)] text-white rounded-[6px] text-[13px] hover:border-[var(--text-muted)] transition-colors">
              {t('Yopish') || 'Close'}
            </button>
          </div>
        </div>
      )}

      {/* Toast Notification */}
      {toast && (
        <div style={{
            position: 'fixed',
            bottom: '80px',
            right: '20px',
            background: 'var(--bg-tertiary)',
            border: '1px solid var(--border)',
            borderLeft: `3px solid ${toast.type === 'error' ? 'var(--danger)' : 'var(--success)'}`,
            color: 'var(--text-primary)',
            padding: '10px 14px',
            borderRadius: '6px',
            fontSize: '13px',
            zIndex: 9999,
            animation: 'fadeIn 0.2s ease'
        }}>
            {toast.message}
        </div>
      )}
    </div>
  );
}

export default App;
