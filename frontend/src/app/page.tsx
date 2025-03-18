"use client"
import Aurora from '../blocks/Backgrounds/Aurora/Aurora';
import GlitchText from '../blocks/TextAnimations/GlitchText/GlitchText';
import VideoUploader from '@/blocks/my_components/VideoUploader';

export default function Home() {
  return (
    <>
   <div className='h-screen w-screen flex flex-col justify-center items-center overflow-hidden'>
     <Aurora
    colorStops={["#04FFFF", "#FFFFFF", "#FF0100"]}
    speed={0.5}
    />
    <VideoUploader />
    <div className="flex flex-col items-center justify-center mb-20 mt-40">
      <GlitchText
        speed={4}
        enableShadows={true}
        enableOnHover={false}
        className='glitch text-2xl text-white'
      >
        Censor AI
      </GlitchText>
    </div>
   </div>
   </>
  );
}
